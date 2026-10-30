import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

const recordId = 'a'.repeat(32);
const postId = 'c'.repeat(32);
const humanReplyId = 'd'.repeat(32);

test('feed surfaces new issues, handoffs, outcomes and blockers while routine replies stay in threads', async (t) => {
  const page = await fixture(t);
  page.setDefaultTimeout(8000);
  await page.evaluate(() => {
    const id = (n: number) => n.toString(16).repeat(32);
    const entries = [
      {id: id(7), actor: 'builder', issue: 42, parent_id: id(2), status: 'implement:finished:succeeded', summary: 'Delivered the working task list.'},
      {id: id(6), actor: 'builder', issue: 42, parent_id: id(2), payload: {communication: {kind: 'blocked', title: 'Storage unavailable', summary: 'Waiting for storage access.'}}},
      {id: id(5), actor: 'builder', issue: 43, parent_id: id(3), status: 'implement:assigned', summary: 'Starting the next issue.'},
      {id: id(4), actor: 'inspector', issue: 42, parent_id: id(3), status: 'review:assigned', summary: 'Starting independent review.'},
      {id: id(3), actor: 'builder', issue: 42, parent_id: id(2), status: 'implement:running', summary: 'Routine progress on the task list.'},
      {id: id(2), actor: 'builder', issue: 42, parent_id: id(1), status: 'implement:assigned', summary: 'Starting the task list.'},
      {id: id(1), actor: 'architect', issue: 42, status: 'investigate:finished:succeeded', summary: 'Plan ready.'},
      // Parent lies outside the page: same-role progress still stays in its thread.
      {id: id(8), actor: 'builder', issue: 42, parent_id: id(9), parent_actor: 'builder', parent_author: 'Builder', parent_issue: 42, status: 'implement:running', summary: 'Off-page routine progress.'},
      // Human actor metadata can be empty; the first role handoff still surfaces.
      {id: id(10), actor: 'issue_maker', parent_id: id(11), parent_actor: '', parent_author: 'alice', status: 'issue_maker:running', summary: 'First Issue Maker handoff.'},
      {id: id(11), actor: '1', author: 'alice', summary: 'Please build a task list.'},
    ].map((entry, index) => ({kind: 'record', repository: 7, seq: 20 - index, author: entry.actor === 'builder' ? 'Builder' : entry.actor === 'architect' ? 'Architect' : 'Inspector', ...entry}));
    window.drawerFixture.setReply(async call => {
      if (call.url.endsWith('/api/factory/feed/subscription')) return Response.json({subscriber: '1', repositories: [7], revision: 1});
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}], next_cursor: ''});
      if (call.url.includes('/api/factory/feed?')) return Response.json({entries: document.body.dataset.humanOffPage === 'true' ? entries.filter(entry => entry.id !== id(11)) : entries, truncated: false});
      if (call.url.includes('/replies?')) {
        const parent = /\/posts\/([a-f0-9]{32})\/replies\?/.exec(call.url)?.[1];
        return Response.json({items: entries.filter(entry => entry.parent_id === parent).map(entry => ({post_id: entry.id, repository: 7, author_name: entry.author, actor: entry.actor, issue_scope: entry.issue, parent_id: entry.parent_id, at: 1729000000, body: entry.summary ?? '', status: entry.status, communication: entry.payload?.communication})), next_before: 0});
      }
      return null;
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Starting the task list.', {exact: true}).waitFor();
  const ids = await feed.locator('article.soda-feed-post').evaluateAll(nodes => nodes.map(node => node.id));
  assert.deepEqual(ids, [7, 6, 5, 4, 2, 1, 10, 11].map(n => 'feed-record-' + n.toString(16).repeat(32)));
  await page.evaluate(async () => {
    document.body.dataset.humanOffPage = 'true';
    await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
  });
  await feed.getByText('First Issue Maker handoff.', {exact: true}).waitFor();
  assert.equal(await feed.getByText('Please build a task list.', {exact: true}).count(), 0);
  assert.equal(await feed.getByText('Routine progress on the task list.', {exact: true}).count(), 0);
  assert.equal(await feed.getByText('Off-page routine progress.', {exact: true}).count(), 0);
  await feed.locator(`[data-feed-drawer-toggle="record:${'2'.repeat(32)}"]`).click();
  await feed.locator('#post-' + '3'.repeat(32)).getByText('Routine progress on the task list.', {exact: true}).waitFor();
  await feed.locator('#post-' + '3'.repeat(32)).locator('[data-feed-drawer-toggle]').click();
  await feed.locator('#post-' + '4'.repeat(32)).getByText('Starting independent review.', {exact: true}).waitFor();
});

test('Issue Maker and its receipts stay separate feed posts connected by parent links', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    document.body.dataset.makerState = 'waiting';
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [
            {
              id: '7',
              owner: 'soda-tester',
              name: 'demo',
              can_create: false,
              project: {id: 'p' + '1'.repeat(24), provisioned: true},
            },
          ],
          next_cursor: '',
        });
      if (call.url.endsWith('/api/factory/feed/posts/' + 'b'.repeat(32)))
        return Response.json({
          post_id: 'b'.repeat(32),
          repository: 7,
          actor: 'issue_maker',
          author_name: 'Issue Maker',
          at: 1729000000,
          status: `issue_maker:${document.body.dataset.makerState}`,
          communication: {
            kind: 'system',
            title: 'Making issues: add login and notifications',
            summary: `Issue Maker is ${document.body.dataset.makerState}.`,
          },
        });
      if (call.url.includes('/replies?'))
        return Response.json({
          items: [12, 13, 14].map((issue, index) => ({
            post_id: String(index + 1).repeat(32),
            repository: 7,
            actor: 'issue_maker',
            author_name: 'Issue Maker',
            at: 1729000000,
            issue_scope: issue,
            status: 'issue_maker:created',
            communication: {kind: 'system', title: `Requested change ${issue}`, summary: `Created issue #${issue}.`},
          })),
        });
      if (!call.url.includes('/api/factory/feed?')) return null;
      if (document.body.dataset.makerOffPage === 'true') return Response.json({entries: [], truncated: false});
      const parent = 'b'.repeat(32);
      return Response.json({
        entries: [
          ...[14, 13, 12].map((issue, index) => ({
            kind: 'post',
            id: String(index + 1).repeat(32),
            repository: 7,
            seq: 5 - index,
            actor: 'issue_maker',
            author: 'Issue Maker',
            issue,
            parent_id: parent,
            parent_author: 'Issue Maker',
            status: 'issue_maker:created',
            payload: {
              communication: {
                kind: 'system',
                title: `Requested change ${issue}`,
                summary: `Created issue #${issue}.`,
                parent_id: parent,
              },
            },
          })),
          {
            kind: 'post',
            id: parent,
            repository: 7,
            seq: 2,
            actor: 'issue_maker',
            author: 'Issue Maker',
            status: `issue_maker:${document.body.dataset.makerState}`,
            payload: {
              communication: {
                kind: 'system',
                title: 'Making issues: add login and notifications',
                summary:
                  document.body.dataset.makerState === 'waiting'
                    ? 'Issues exist. AI refinement is waiting for preparation.'
                    : 'Issue Maker run has started. Refining the recorded issues.',
              },
            },
          },
          {
            kind: 'post',
            id: 'c'.repeat(32),
            repository: 7,
            seq: 1,
            author: 'alice',
            post: {
              post_id: 'c'.repeat(32),
              repository: 7,
              revision: 0,
              author: '1',
              author_name: 'alice',
              body: 'Add login and notifications',
              at: 1729000000,
              intake: 'recorded',
              judgment: 'applied',
              route: 'new_issue',
              draft: 'applied',
              effect: 'verified',
              issue: 12,
              issue_maker: {post: parent, state: document.body.dataset.makerState},
            },
          },
        ],
        truncated: false,
      });
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const maker = feed.locator('#feed-post-' + 'b'.repeat(32));
  await maker.getByText('AI refinement is waiting for preparation.', {exact: false}).waitFor();
  assert.equal(await maker.count(), 1);
  assert.equal(await maker.getAttribute('data-state'), 'waiting');
  assert.equal(await maker.locator('[aria-label="working"]').count(), 0);
  assert.equal(await feed.locator('article[data-role="issue_maker"]').count(), 4);
  for (const [index, issue] of [14, 13, 12].entries()) {
    const receipt = feed.locator('#feed-post-' + String(index + 1).repeat(32));
    assert.match(
      (await receipt.getByRole('link', {name: `Open issue #${issue}`, exact: true}).getAttribute('href'))!,
      new RegExp(`/issues/${issue}$`)
    );
    assert.equal(await receipt.locator('.soda-feed-parent').innerText(), 'Replying to Issue Maker');
    assert.match((await receipt.locator('.soda-feed-parent a').getAttribute('href'))!, /post=b{32}$/);
    assert.equal(await receipt.locator('article').count(), 0);
  }
  const handoff = feed.getByRole('link', {name: 'Follow Issue Maker'});
  assert.match((await handoff.getAttribute('href'))!, /\/feed\?post=b{32}$/);
  assert.equal(await feed.locator('.soda-feed-outcome').getByText('Product Lead', {exact: true}).count(), 0);
  await page.evaluate(async () => {
    document.body.dataset.makerState = 'running';
    await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
  });
  await page.waitForFunction(
    () => document.querySelector('#feed-post-' + 'b'.repeat(32))?.getAttribute('data-state') === 'in_progress'
  );
  assert.equal(await maker.count(), 1);
  assert.equal(await feed.locator('article[data-role="issue_maker"]').count(), 4);
  await page.evaluate(() => {
    document.body.dataset.makerState = 'waiting';
    (document.querySelector('soda-factory-feed') as unknown as {post: string}).post = 'b'.repeat(32);
  });
  await maker.getByText('Issue Maker is waiting.', {exact: true}).waitFor();
  assert.equal(await maker.getAttribute('data-state'), 'waiting');
  for (const issue of [12, 13, 14]) await maker.getByRole('link', {name: new RegExp(`issue #${issue}:`)}).waitFor();
  // A focused post keeps updating even after it falls outside the newest feed page.
  await page.evaluate(async () => {
    document.body.dataset.makerOffPage = 'true';
    document.body.dataset.makerState = 'running';
    await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
  });
  await maker.getByText('Issue Maker is running.', {exact: true}).waitFor();
  assert.equal(await maker.getAttribute('data-state'), 'in_progress');
  assert.equal(await maker.count(), 1);
  for (const issue of [12, 13, 14])
    assert.match(
      (await maker.getByRole('link', {name: new RegExp(`issue #${issue}:`)}).getAttribute('href'))!,
      new RegExp(`/issues/${issue}$`)
    );
});

test('new-work authoring shows waiting, queued, started and settled states without false working indicators', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    document.body.dataset.authoringState = 'waiting';
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [
            {
              id: '7',
              owner: 'soda-tester',
              name: 'demo',
              can_create: false,
              project: {id: 'p' + '1'.repeat(24), provisioned: true},
            },
          ],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) {
        const state = document.body.dataset.authoringState;
        return Response.json({
          entries: [
            {
              kind: 'post',
              id: 'c'.repeat(32),
              repository: 7,
              seq: 1,
              author: 'alice',
              summary: 'Please add a login flow',
              post: {
                post_id: 'c'.repeat(32),
                repository: 7,
                revision: 0,
                author: '1',
                author_name: 'alice',
                body: 'Please add a login flow',
                at: 1729000000,
                intake: 'recorded',
                judgment: 'applied',
                route: 'new_issue',
                draft: state === 'created' ? 'applied' : 'pending',
                effect: state === 'created' ? 'verified' : 'pending',
                issue: state === 'created' ? 12 : 0,
                ...(state === 'created'
                  ? {}
                  : {
                      authoring: {
                        state,
                        reason: 'preparation_not_ready',
                        detail: state === 'waiting' ? 'project has no accepted requirements' : '',
                        intent: 'd' + 'b'.repeat(24),
                        run: state === 'running' ? 'recorded-run' : '',
                        since_unix: 1729000000,
                        checked_unix: 1729000010,
                      },
                    }),
              },
            },
          ],
          truncated: false,
        });
      }
      return null;
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const outcome = feed.locator('.soda-feed-outcome');
  await outcome.getByText('project has no accepted requirements', {exact: true}).waitFor();
  assert.equal(await outcome.getAttribute('data-phase'), 'draft_waiting');
  await outcome.getByText('Factory', {exact: true}).waitFor();
  await outcome.getByText('Waiting', {exact: true}).waitFor();
  await outcome.getByText(/Last checked/).waitFor();
  assert.equal(await outcome.locator('[aria-label="working"]').count(), 0);
  assert.match(
    (await outcome.getByRole('link', {name: 'View request'}).getAttribute('href'))!,
    /\/requests\/d[b]{24}$/
  );
  assert.match(
    (await outcome.getByRole('link', {name: 'Review Project setup'}).getAttribute('href'))!,
    /\/settings\/factory$/
  );
  for (const [state, phase, working] of [
    ['queued', 'draft_queued', 0],
    ['running', 'drafting', 1],
    ['settling', 'draft_settling', 0],
    ['failed', 'draft_failed', 0],
    ['unknown', 'draft_unknown', 0],
    ['created', 'created', 0],
  ] as const) {
    await page.evaluate(async (state) => {
      document.body.dataset.authoringState = state;
      await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
    }, state);
    await page.waitForFunction(
      (phase) => document.querySelector('.soda-feed-outcome')?.getAttribute('data-phase') === phase,
      phase
    );
    assert.equal(await outcome.locator('[aria-label="working"]').count(), working, state);
  }
  await outcome.getByRole('link', {name: 'Open issue #12'}).waitFor();
});

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
  const draft = feed.getByLabel('New post', {exact: true});
  await draft.fill('Keep this unsent note.');
  await page.evaluate(() => {
    document.body.dataset.feedUnavailable = 'false';
  });
  await feed.getByRole('button', {name: 'Retry loading activity', exact: true}).click();
  await feed.locator('.soda-feed-status').getByText('Decided', {exact: true}).waitFor();
  assert.equal(await feed.locator('.soda-feed-status').getByText('Done', {exact: true}).count(), 2);
  assert.equal(await feed.locator('.soda-feed-status').getByText('For your information', {exact: true}).count(), 1);
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  await feed.getByRole('button', {name: 'Choices & approvals', exact: true}).click();
  await feed.getByRole('heading', {name: 'No matching posts.', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Clear filters', exact: true}).click();
  await feed.locator('.soda-feed-status').getByText('Decided', {exact: true}).waitFor();
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

test('Project feed reads newest-first with enriched posts and live comment outcomes from one read', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
  });
  await page.evaluate(
    async ({recordId, postId, humanReplyId}) => {
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
                status: 'route:discussion',
                post: {
                  post_id: postId,
                  revision: 0,
                  repository: 7,
                  author: '1',
                  author_name: 'Ada',
                  at: 1720000000,
                  edited: false,
                  body: 'Should the demo cover substitute approvers?',
                  intake: 'recorded',
                  judgment: 'applied',
                  route: 'discussion',
                  draft: 'pending',
                  effect: 'pending',
                  issue: 0,
                },
              },
            ],
            next: 'v1:41',
            truncated: true,
          });
        }
        if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST') {
          const body = JSON.parse((call as unknown as {body?: string}).body ?? '{}') as Record<string, unknown>;
          state.comments.push(typeof body.body === 'string' ? body.body : '');
          return Response.json({
            post_id: humanReplyId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        }
        if (call.url.includes('/replies?') && !call.url.includes(`/posts/${recordId}/replies?`))
          return Response.json({items: [], next_before: 0});
        if (call.url.includes('/replies?'))
          return Response.json({
            items:
              state.comments.length === 0
                ? []
                : [
                    {
                      post_id: humanReplyId,
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
                    },
                  ],
            next_before: 0,
          });
        return null;
      });
    },
    {recordId, postId, humanReplyId}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  // Separate posts with the outside-card author context and stored text.
  await feed.getByText('Recorded work').first().waitFor();
  await feed.getByText('Ada').first().waitFor();
  await feed.locator('.soda-feed-header').getByText('Product Lead').first().waitFor();
  await feed.locator('.soda-feed-post[data-card=post]').first().waitFor();
  await feed
    .getByText('This reads as a discussion, so I’ve left it here for people to answer.', {exact: true})
    .waitFor();
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
  await feed.locator(`[data-post-submit="record:${recordId}"]`).click();
  await feed.getByText('Reply saved.').waitFor();
  const posted = await page.evaluate(() => {
    const drawer = window.drawerFixture as unknown as {calls: {url: string; method: string; body?: string}[]};
    return drawer.calls
      .filter((call) => call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST')
      .map((call) => JSON.parse(call.body ?? '{}'));
  });
  assert.equal(posted.length, 1);
  assert.equal(posted[0]?.repository, 7);
  // Commenting from a work card carries its authorized bound source issue
  // and names the source entry for server-side resolution.
  assert.equal(posted[0]?.issue, 3);
  assert.equal(posted[0]?.parent_id, recordId);
  assert.equal('source_id' in posted[0], false);
  assert.equal(posted[0]?.body, 'half-written reply');
  assert.match(
    typeof posted[0]?.idempotency_key === 'string' ? (posted[0]?.idempotency_key as string) : '',
    /^[A-Za-z0-9_-]{1,64}$/
  );
  assert.equal('author' in (posted[0] ?? {}), false);
  // The refresh after saving re-reads progress without another submission.
  const issueLink = feed.getByRole('link', {name: 'Open issue #12'});
  await issueLink.waitFor();
  assert.equal(await issueLink.getAttribute('href'), '/soda-tester/demo/issues/12');
  await feed.getByText(/I created issue #12 for this\./).waitFor();
  assert.equal(
    await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.url.includes('/feed/comments/')).length),
    0
  );
});

test('a project comment works before the first worker post', async (t) => {
  const page = await fixture(t);
  await page.evaluate(
    ({humanReplyId}) => {
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: [], revision: 1});
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        const posted = window.drawerFixture.calls.some(
          (c) => c.url.endsWith('/api/factory/feed/posts') && c.method === 'POST'
        );
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: posted ? [{kind: 'post', id: humanReplyId, repository: 7, seq: 1, post: stored}] : [],
            truncated: false,
          });
        if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST')
          return Response.json({
            post_id: humanReplyId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        return null;
      });
      const stored = {
        post_id: humanReplyId,
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
      };
      window.drawerFixture.mountFeed();
    },
    {humanReplyId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('No activity yet.').waitFor();
  await feed.getByLabel('New post', {exact: true}).fill('Thanks for the update.');
  await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
  await feed.getByText('This reads as a discussion, so I’ve left it here for people to answer.').waitFor();
  assert.equal(await feed.getByLabel('New post', {exact: true}).inputValue(), '');
  assert(await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).isDisabled());
  const posts = await page.evaluate(() =>
    window.drawerFixture.calls
      .filter((c) => c.url.endsWith('/api/factory/feed/posts') && c.method === 'POST')
      .map((c) => JSON.parse(c.body ?? '{}'))
  );
  assert.equal(posts.length, 1);
  assert.equal(posts[0].repository, 7);
  assert.equal(posts[0].body, 'Thanks for the update.');
  assert.equal('source_id' in posts[0], false);
  assert.equal('parent_id' in posts[0], false);
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
    () =>
      !document.querySelector<HTMLButtonElement>(
        'soda-factory-feed[data-feed-race="true"] button[aria-label="Refresh feed"]'
      )?.disabled
  );
  await feed.evaluate((el) => (el as HTMLElement & {releaseFirst(): void}).releaseFirst());
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  assert.ok((await feed.evaluate((el) => (el as HTMLElement & {readCount(): number}).readCount())) >= 3);
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
});

test('an uncertain comment retry resends the original payload and keeps newer writing', async (t) => {
  const page = await fixture(t);
  await page.evaluate(
    ({humanReplyId}) => {
      let attempts = 0;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST') {
          attempts++;
          return attempts === 1
            ? Response.json({error: {code: 'reply_lost'}}, {status: 503})
            : Response.json({
                post_id: humanReplyId,
                revision: 0,
                intake: 'recorded',
                judgment: 'applied',
                route: 'discussion',
                draft: 'pending',
                effect: 'pending',
                issue: 0,
              });
        }
        if (call.url.endsWith('/api/factory/feed/posts/' + humanReplyId))
          return Response.json({
            post_id: humanReplyId,
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
    {humanReplyId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('No activity yet.').waitFor();
  const draft = feed.getByLabel('New post', {exact: true});
  await draft.fill('Original request');
  await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
  await feed.locator('[data-feed-message]').filter({hasText: 'draft is kept'}).waitFor();
  await draft.fill('New writing after the uncertain reply');
  await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
  await feed.getByText('Previous post saved. Your newer edits are kept, ready to send.', {exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'New writing after the uncertain reply');
  const posts = await page.evaluate(() =>
    window.drawerFixture.calls
      .filter((call) => call.method === 'POST' && call.url.endsWith('/api/factory/feed/posts'))
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
      const detail = {
        post_id: clarifyId,
        revision: 3,
        repository: 7,
        author_name: 'alice',
        at: 1720000000,
        edited: false,
        body: 'Original request text',
        intake: 'recorded',
        judgment: 'applied',
        draft: 'pending',
        effect: 'pending',
        route: 'clarify',
        detail: 'What should change, and where should it apply?',
        can_retry: true,
        issue: 0,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.includes('/replies?')) return Response.json({items: [detail], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/posts/${clarifyId}`) && call.method === 'GET')
          return Response.json(detail);
        if (call.url.endsWith(`/api/factory/feed/posts/${clarifyId}/retry`) && call.method === 'POST')
          return Response.json({ok: true});
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, clarifyId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Answer clarification', exact: true}).click();
  const answer = feed.locator('[data-revision-body]');
  await answer.waitFor();
  await answer.fill('My saved answer');
  const draftKey = `soda-feed-revision:v2:fixture-generation:7:${clarifyId}`;
  assert.deepEqual(await page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) ?? 'null'), draftKey), {
    body: 'My saved answer',
    original: 'Original request text',
  });
  // A reload drops the panel; the stored answer survives it.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Answer clarification', exact: true}).click();
  await feed.locator('.soda-feed-revision').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'My saved answer');
  assert.equal(
    await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length),
    0,
    'restoring a saved answer must not write'
  );
  await feed.getByRole('button', {name: 'Submit answer', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posted = await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method === 'POST'));
  assert.deepEqual(JSON.parse(posted[0]?.body ?? '{}'), {
    expected_revision: 3,
    body: 'Original request text\n\nClarification: My saved answer',
  });
  assert.equal(await page.evaluate((key) => window.localStorage.getItem(key), draftKey), null);
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
});

test('posts and replies edit their own identities and a reply can receive another reply', async (t) => {
  const page = await fixture(t);
  page.setDefaultTimeout(8000);
  const root = 'a'.repeat(32), child = 'b'.repeat(32), nested = 'c'.repeat(32);
  await page.evaluate(({root, child, nested}) => {
    const posts: Record<string, Record<string, unknown>> = {};
    const make = (id: string, body: string, parent = '') => ({
      post_id: id, repository: 7, revision: 0, author: '1', author_name: 'alice',
      at: 1720000000, body, parent_id: parent, intake: 'recorded', judgment: 'applied',
      route: 'discussion', draft: 'pending', effect: 'pending', can_retry: true,
    });
    posts[root] = make(root, 'Root post');
    posts[child] = make(child, 'Child reply', root);
    window.drawerFixture.setReply(async call => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: [], revision: 1});
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
      if (call.url.includes('/api/factory/feed?')) return Response.json({
        entries: [{kind: 'post', id: root, repository: 7, seq: 1, post: posts[root]}], truncated: false,
      });
      if (call.url.includes('/replies?')) {
        const parent = /\/posts\/([a-f0-9]{32})\/replies\?/.exec(call.url)?.[1];
        return Response.json({items: Object.values(posts).filter(post => post.parent_id === parent).reverse(), next_before: 0});
      }
      if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST') {
        const input = JSON.parse(call.body ?? '{}');
        posts[nested] = make(nested, input.body, input.parent_id);
        return Response.json(posts[nested]);
      }
      const match = /\/factory\/feed\/posts\/([a-f0-9]{32})(\/retry)?$/.exec(call.url);
      if (match && posts[match[1]!]) {
        const post = posts[match[1]!]!;
        if (match[2] && call.method === 'POST') {
          post.body = JSON.parse(call.body ?? '{}').body;
          post.revision = Number(post.revision) + 1;
        }
        return Response.json(post);
      }
      return null;
    });
    window.drawerFixture.mountFeed();
  }, {root, child, nested});
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Root post', {exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
  const editor = feed.getByRole('region', {name: 'Edit post', exact: true});
  await editor.getByLabel('Edit your post', {exact: true}).fill('Updated root post');
  await editor.getByRole('button', {name: 'Save post and retry', exact: true}).click();
  await feed.getByText('Updated root post', {exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="post:${root}"]`).click();
  const childPost = feed.locator('#post-' + child);
  await childPost.getByText('Child reply', {exact: true}).waitFor();
  await childPost.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  const replyEditor = feed.getByRole('region', {name: 'Edit reply', exact: true});
  await replyEditor.getByLabel('Edit your reply', {exact: true}).fill('Updated child reply');
  await replyEditor.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await childPost.getByText('Updated child reply', {exact: true}).waitFor();
  await feed.getByText('Updated root post', {exact: true}).waitFor();
  await childPost.locator('[data-reply-compose]').click();
  await childPost.getByLabel('Reply to alice', {exact: true}).fill('A reply to the child');
  await childPost.locator(`[data-post-submit="post:${child}"]`).click();
  const nestedPost = feed.locator('#post-' + nested);
  await nestedPost.getByText('A reply to the child', {exact: true}).waitFor();
  assert.equal(await nestedPost.locator('.soda-feed-reply-parent a').getAttribute('href'), '#post-' + child);
  const writes = await page.evaluate(() => window.drawerFixture.calls.filter(call => call.method === 'POST'));
  assert(writes[0]?.url.endsWith('/' + root + '/retry'));
  assert(writes[1]?.url.endsWith('/' + child + '/retry'));
  const posted = JSON.parse(writes[2]?.body ?? '{}');
  assert.equal(posted.parent_id, child);
  assert.equal('source_id' in posted, false);
  assert.doesNotMatch(await feed.innerText(), /\bcomments?\b/i);
  const accessibleLabels = await feed.locator('[aria-label], [title], [placeholder]').evaluateAll(nodes =>
    nodes.flatMap(node => ['aria-label', 'title', 'placeholder'].map(name => node.getAttribute(name) ?? '')).join('\n')
  );
  assert.doesNotMatch(accessibleLabels, /\bcomments?\b/i);
});

test('revision draft keeps text on failure, shows the newer request on conflict, discards explicitly', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const reviseId = 'e'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const state = {revision: 5, failNext: false};
      (window as unknown as {revState: typeof state}).revState = state;
      const detail = () => ({
        post_id: reviseId,
        revision: state.revision,
        repository: 7,
        author_name: 'alice',
        at: 1720000000,
        edited: false,
        body: state.revision === 5 ? 'Original request v5' : 'Original request v6 changed',
        intake: 'recorded',
        judgment: 'applied',
        draft: 'pending',
        effect: 'pending',
        route: 'discussion',
        can_retry: true,
        issue: 0,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.includes('/replies?')) return Response.json({items: [detail()], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/posts/${reviseId}`) && call.method === 'GET')
          return Response.json(detail());
        if (call.url.endsWith(`/api/factory/feed/posts/${reviseId}/retry`) && call.method === 'POST') {
          if (state.failNext)
            return Response.json({error: {message: 'Temporary failure. Your edits are kept.'}}, {status: 503});
          if (JSON.parse(call.body ?? '{}').expected_revision !== state.revision)
            return Response.json({error: {message: 'stale revision'}}, {status: 409});
          state.revision++;
          return Response.json({post: reviseId, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  const setState = (next: {revision?: number; failNext?: boolean}) =>
    page.evaluate((next) => Object.assign((window as unknown as {revState: object}).revState, next), next);
  const draftKey = `soda-feed-revision:v2:fixture-generation:7:${reviseId}`;
  const storedBody = () =>
    page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) ?? 'null')?.body ?? null, draftKey);
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  assert.equal(await body.inputValue(), 'Original request v5');
  await body.fill('Revised request kept');
  assert.equal(await storedBody(), 'Revised request kept');
  await setState({failNext: true});
  await feed.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await feed.locator('[data-feed-message]').filter({hasText: 'kept'}).waitFor();
  assert.equal(await body.inputValue(), 'Revised request kept');
  assert.equal(await storedBody(), 'Revised request kept');
  // The request changes elsewhere while the panel is open: the stale submit
  // is refused, the newer text shows beside the kept draft, and the next
  // submit goes against the newer revision.
  await setState({failNext: false, revision: 6});
  await feed.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await feed
    .locator('.soda-feed-revision blockquote')
    .getByText('Original request v6 changed', {exact: true})
    .waitFor();
  assert.equal(await body.inputValue(), 'Revised request kept');
  await feed.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const retries = await page.evaluate(() =>
    window.drawerFixture.calls
      .filter((call) => call.url.endsWith('/retry'))
      .map((call) => JSON.parse(call.body ?? '{}'))
  );
  assert.deepEqual(
    retries.map((retry) => retry.expected_revision),
    [5, 5, 6]
  );
  assert.equal(await storedBody(), null);
  // Cancel discards kept text; the next open serves the server text.
  await feed.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  await body.waitFor();
  await body.fill('Abandoned edit');
  await feed.getByRole('button', {name: 'Cancel', exact: true}).click();
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
  assert.equal(await storedBody(), null);
  await feed.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  await body.waitFor();
  assert.equal(await body.inputValue(), 'Original request v6 changed');
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
        if (call.url.includes('/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (before)
            return Response.json({
              items: [
                {
                  post_id: target,
                  repository: 7,
                  revision: 0,
                  actor: 'architect',
                  author_name: 'Architect',
                  assignment: work,
                  parent_id: invite,
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
                post_id: invite,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                parent_id: '',
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
  await feed.locator(`#post-${target}`).getByText('The role’s actual reply from the older page.').waitFor();
  assert.equal(await page.evaluate(() => location.hash), '');
  await page.waitForFunction((id) => document.activeElement?.id === id, `post-${target}`);
  assert.ok(
    (await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.url.includes('/replies?') && call.url.includes('before=')
        ).length
    )) >= 1,
    'the older replies page is read'
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('older thread pages prepend and refreshes preserve canonical reply order', async (t) => {
  const page = await fixture(t);
  const work = '2'.repeat(32);
  // Canonical seq order is oldest, middle, newest; oldest and middle share
  // one Unix second across the page boundary. Ids are chosen so a
  // humanReplyId tiebreak would misorder them (newest sorts first).
  const oldest = 'f'.repeat(32);
  const middle = 'a'.repeat(32);
  const newest = '5'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, oldest, middle, newest, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const discussion = (post_id: string, author_name: string, body: string, stamp: number) => ({
        post_id,
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
        if (call.url.includes('/replies?')) {
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
      .locator('.soda-feed-thread li.soda-feed-reply')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#post-${middle}`).getByText('Covered in the candidate.').waitFor();
  assert.deepEqual(await orderOf(), [`post-${middle}`, `post-${newest}`]);
  // Older-page prepend keeps ascending canonical order across the
  // same-second page boundary.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#post-${oldest}`).getByText('Can this cover substitute approvers?').waitFor();
  assert.deepEqual(await orderOf(), [`post-${oldest}`, `post-${middle}`, `post-${newest}`]);
  // A quiet feed refresh re-merges the newest page; dedup keeps every
  // canonical position instead of reordering.
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.').waitFor();
  assert.deepEqual(await orderOf(), [`post-${oldest}`, `post-${middle}`, `post-${newest}`]);
  // Reload replace re-reads the newest page, then paging restores the
  // older reply in front again.
  await feed.getByRole('button', {name: 'Refresh replies'}).click();
  await feed.locator('.soda-feed-thread').getByText('Covered in the candidate.').waitFor();
  assert.deepEqual(await orderOf(), [`post-${middle}`, `post-${newest}`]);
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#post-${oldest}`).getByText('Can this cover substitute approvers?').waitFor();
  assert.deepEqual(await orderOf(), [`post-${oldest}`, `post-${middle}`, `post-${newest}`]);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

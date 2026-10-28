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
  await feed.locator('.soda-feed-status').getByText('Decided', {exact: true}).waitFor();
  assert.equal(await feed.locator('.soda-feed-status').getByText('Done', {exact: true}).count(), 2);
  assert.equal(await feed.locator('.soda-feed-status').getByText('For your information', {exact: true}).count(), 1);
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  await feed.getByRole('button', {name: 'Decisions', exact: true}).click();
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
                status: 'route:discussion',
                comment: {
                  comment_id: postId,
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
        if (call.url.includes('/api/factory/feed/replies?') && !call.url.includes(`source_id=${recordId}`))
          return Response.json({items: [], next_before: 0});
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items:
              state.comments.length === 0
                ? []
                : [
                    {
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
                    },
                  ],
            next_before: 0,
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
  await feed.locator('.soda-feed-status').getByText('Comment', {exact: true}).waitFor();
  await feed.getByText('Kept as discussion.', {exact: true}).waitFor();
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
  await feed.getByText(/Issue #12 created\./).waitFor();
  assert.equal(
    await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.url.includes('/feed/comments/')).length),
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
        const posted = window.drawerFixture.calls.some(
          (c) => c.url.endsWith('/api/factory/feed/comments') && c.method === 'POST'
        );
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: posted ? [{kind: 'post', id: commentId, repository: 7, seq: 1, comment: stored}] : [],
            truncated: false,
          });
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST')
          return Response.json({
            comment_id: commentId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        return null;
      });
      const stored = {
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
      };
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
      const detail = {
        comment_id: clarifyId,
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
        if (call.url.includes('/api/factory/feed/replies?')) return Response.json({items: [detail], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/comments/${clarifyId}`) && call.method === 'GET')
          return Response.json(detail);
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
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
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

test('revision draft keeps text on failure, shows the newer request on conflict, discards explicitly', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const reviseId = 'e'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const state = {revision: 5, failNext: false};
      (window as unknown as {revState: typeof state}).revState = state;
      const detail = () => ({
        comment_id: reviseId,
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
        if (call.url.includes('/api/factory/feed/replies?')) return Response.json({items: [detail()], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json(detail());
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.failNext)
            return Response.json({error: {message: 'Temporary failure. Your edits are kept.'}}, {status: 503});
          if (JSON.parse(call.body ?? '{}').expected_revision !== state.revision)
            return Response.json({error: {message: 'stale revision'}}, {status: 409});
          state.revision++;
          return Response.json({comment: reviseId, revision: state.revision});
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
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  assert.equal(await body.inputValue(), 'Original request v5');
  await body.fill('Revised request kept');
  assert.equal(await storedBody(), 'Revised request kept');
  await setState({failNext: true});
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.locator('[data-feed-message]').filter({hasText: 'kept'}).waitFor();
  assert.equal(await body.inputValue(), 'Revised request kept');
  assert.equal(await storedBody(), 'Revised request kept');
  // The request changes elsewhere while the panel is open: the stale submit
  // is refused, the newer text shows beside the kept draft, and the next
  // submit goes against the newer revision.
  await setState({failNext: false, revision: 6});
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed
    .locator('.soda-feed-revision blockquote')
    .getByText('Original request v6 changed', {exact: true})
    .waitFor();
  assert.equal(await body.inputValue(), 'Revised request kept');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
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
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await body.waitFor();
  await body.fill('Abandoned edit');
  await feed.getByRole('button', {name: 'Cancel', exact: true}).click();
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
  assert.equal(await storedBody(), null);
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await body.waitFor();
  assert.equal(await body.inputValue(), 'Original request v6 changed');
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

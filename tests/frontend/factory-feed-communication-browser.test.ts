import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh} from './fixtures/project-controls-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';

setupProjectControlsDriver();

test('a Finance comment opens an explicit change review and reopens its draft without another mutation', async (t) => {
  const page = await fixture(t),
    work = 'e'.repeat(32),
    comment = 'a'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, comment, intent}) => {
      let reviewed = false,
        reads = 0;
      window.drawerFixture.api.setPresentation('settings');
      document.body.addEventListener('soda-intent-open', (event) => {
        document.body.dataset.changeOpened = JSON.stringify((event as CustomEvent).detail);
        document.body.dataset.changeOpenCount = String(Number(document.body.dataset.changeOpenCount ?? '0') + 1);
      });
      const change = () => ({
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        ...(reviewed ? {draft_revision: 'd' + 'd'.repeat(24)} : {}),
        state: reads > 2 ? 'superseded' : reviewed ? 'draft' : 'review_required',
        can_review: true,
        verified: false,
        evidence_status: 'unavailable',
      });
      const detail = () => ({
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'recorded',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        body: 'New requests above EUR 1000 require Finance.',
        author_name: 'Maya',
        at: 1729000000,
        product_change: change(),
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
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items: [
              detail(),
              {
                ...detail(),
                comment_id: 'f'.repeat(32),
                product_change: {...change(), state: 'review_required', can_review: false},
              },
            ],
            next_before: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}/review`) && call.method === 'POST') {
          reviewed = true;
          return Response.json(detail());
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`)) {
          reads++;
          return Response.json(detail());
        }
        return null;
      });
    },
    {work, comment, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  await feed.getByText('Waiting for the request owner to review this proposed change.', {exact: true}).waitFor();
  const draft = feed.locator(`textarea[data-feed-draft="record:${work}"]`);
  await draft.fill('Keep the delivered EUR 420 request and its history.');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
  await feed.getByRole('button', {name: 'Continue change review', exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => JSON.parse(document.body.dataset.changeOpened ?? '{}')), {
    id: intent,
    repository: '7',
  });
  const writes = await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET'));
  assert.equal(writes.length, 1);
  assert.deepEqual(JSON.parse(writes[0]!.body ?? '{}'), {revision: 0});
  assert.equal(writes[0]!.url.endsWith(`/api/factory/feed/comments/${comment}/review`), true);
  await feed.getByRole('button', {name: 'Continue change review', exact: true}).click();
  await page.waitForFunction(() => document.body.dataset.changeOpenCount === '2');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 1);
  await feed.getByRole('button', {name: 'Continue change review', exact: true}).click();
  await feed
    .getByText(
      'This change could not be opened. Refresh the conversation to check its current direction and your access; the comment is kept.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.evaluate(() => document.body.dataset.changeOpenCount), '2');
  assert.equal(await draft.inputValue(), 'Keep the delivered EUR 420 request and its history.');
});

test('actual consultation states refresh in the work thread while filtered views preserve the human draft', async (t) => {
  const page = await fixture(t),
    work = 'a'.repeat(32),
    call = 'b'.repeat(32),
    decision = 'c'.repeat(32);
  await page.evaluate(
    ({work, call, decision}) => {
      let reads = 0;
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (request) => {
        if (request.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (request.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (request.url.includes('/api/factory/feed?')) {
          const filtered = new URL(request.url, 'http://localhost').searchParams.get('view') === 'decisions';
          return Response.json({
            entries: filtered
              ? [
                  {
                    kind: 'post',
                    id: decision,
                    repository: 7,
                    seq: 2,
                    actor: 'factory',
                    author: 'Factory',
                    payload: {
                      communication: {
                        kind: 'system',
                        component_id: 'F11',
                        title: 'Finance gate retained',
                        summary: 'Finance approval is required above EUR 1000.',
                        decision: 'Require Finance',
                        rationale: 'Recorded product rule.',
                        plan_source: {
                          attempt_root: work,
                          plan: decision,
                          digest: 'e'.repeat(64),
                          source_commit: 'f'.repeat(40),
                          generation: 1,
                        },
                      },
                    },
                  },
                  {
                    kind: 'post',
                    id: 'd'.repeat(32),
                    repository: 7,
                    seq: 3,
                    actor: 'factory',
                    author: 'Factory',
                    payload: {
                      communication: {
                        kind: 'system',
                        component_id: 'F11',
                        title: 'Implementation strategy recorded',
                        summary: 'Implement accepted outcomes in increments.',
                        planning_source: {
                          snapshot: 'a'.repeat(64),
                          record_digest: 'b'.repeat(64),
                          strategy: 'incremental_acceptance',
                          outcome: 'planned',
                          provisional: false,
                          recorded_unix: 1729000000,
                          judgment: {
                            selected: 'direct_small_change',
                            model: 'recorded-model',
                            confidence: 0.4,
                            probabilities: {direct_small_change: 0.6, incremental_acceptance: 0.4},
                            input_tokens: 20,
                            output_tokens: 10,
                          },
                        },
                      },
                    },
                  },
                ]
              : [
                  {
                    kind: 'record',
                    id: work,
                    repository: 7,
                    seq: 1,
                    actor: 'builder',
                    author: 'Builder',
                    status: 'assigned',
                    summary: 'Working on the equipment request app.',
                  },
                ],
            truncated: false,
          });
        }
        if (request.url.includes('/api/factory/feed/replies?')) {
          const returned = reads++ > 0;
          return Response.json({
            items: [
              {
                comment_id: call,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: '',
                body: 'Consulted JEV · choice',
                at: 1729000000,
                consultation: {
                  snapshot: 'd'.repeat(64),
                  group_id: 'd'.repeat(64),
                  assignment: work,
                  tool: 'systemone.choice',
                  kind: 'choice',
                  status: returned ? 'returned' : 'pending',
                  state: {amount: 1001, rule: 'Finance above EUR 1000'},
                  instructions: 'Does this request need Finance approval?',
                  criteria: {yes: 'Finance required', no: 'Manager only'},
                  created_unix: 1729000000,
                  updated_unix: 1729000001,
                  ...(returned
                    ? {
                        model: 'jev-recorded',
                        answer: {
                          answer: {
                            type: 'choice',
                            choice: 'yes',
                            probabilities: {yes: 0.96, no: 0.04},
                            confidence: 0.92,
                          },
                        },
                      }
                    : {}),
                },
              },
            ],
            next_before: 0,
          });
        }
        return null;
      });
    },
    {work, call, decision}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByText('Waiting for JEV', {exact: true}).waitFor();
  const draft = feed.locator(`textarea[data-feed-draft="record:${work}"]`);
  await draft.fill('Keep the existing EUR 420 request after this update.');
  await feed.getByRole('button', {name: 'Refresh replies', exact: true}).click();
  await feed.getByText('JEV returned a result', {exact: true}).waitFor();
  await feed.getByText('Selected: yes', {exact: false}).waitFor();
  assert.equal(await draft.inputValue(), 'Keep the existing EUR 420 request after this update.');
  await feed.getByRole('button', {name: 'Decisions', exact: true}).click();
  await feed.getByRole('heading', {name: 'Finance gate retained', exact: true}).waitFor();
  await feed.getByText('Recorded plan · generation 1', {exact: true}).click();
  await feed.getByText('e'.repeat(64), {exact: true}).waitFor();
  await feed.getByText('Recording this revision does not itself admit or implement the plan.', {exact: true}).waitFor();
  await feed.getByText('Recorded strategy', {exact: true}).click();
  await feed.getByText('Controller choice: incremental_acceptance · planned', {exact: true}).waitFor();
  await feed.getByText('Actual JEV answer', {exact: true}).click();
  assert.match(await feed.locator('.soda-feed-source-reference pre').innerText(), /"selected": "direct_small_change"/);
  await feed.getByRole('button', {name: 'All', exact: true}).click();
  await feed.getByText('JEV returned a result', {exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'Keep the existing EUR 420 request after this update.');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a failed consultation keeps scope-accurate copy while JEV result and explanation coexist across refresh', async (t) => {
  const page = await fixture(t),
    work = 'b'.repeat(32),
    failCall = 'c'.repeat(32),
    okCall = 'd'.repeat(32),
    replyCall = 'e'.repeat(32),
    // Production shape: a consultation record always carries
    // "Consulted JEV" + kind; the worker interpretation arrives as a
    // separate factory.reply with reply_to set (live_test.go:849-864).
    replyTitle = 'Approval research recorded',
    replySummary =
      'I retained all three actual judgments with the supplied policy. My next action is to cite the policy in the approval design review; this research does not approve the request or implement a change.';
  await page.evaluate(
    ({work, failCall, okCall, replyCall, replyTitle, replySummary}) => {
      let reads = 0;
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (request) => {
        if (request.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (request.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (request.url.includes('/api/factory/feed?'))
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
                summary: 'Working on the equipment request app.',
              },
            ],
            truncated: false,
          });
        if (request.url.includes('/api/factory/feed/replies?')) {
          const returned = reads++ > 0;
          const failed = {
            comment_id: failCall,
            repository: 7,
            revision: 0,
            actor: 'builder',
            author_name: 'Builder',
            assignment: work,
            reply_to: '',
            body: 'Consulted JEV · choice',
            at: 1729000000,
            consultation: {
              snapshot: 'a'.repeat(64),
              group_id: 'c'.repeat(64),
              assignment: work,
              tool: 'systemone.choice',
              kind: 'choice',
              status: 'failed',
              state: {amount: 1001, rule: 'Finance above EUR 1000'},
              instructions: 'Does this request need Finance approval?',
              criteria: {yes: 'Finance required', no: 'Manager only'},
              created_unix: 1729000000,
              updated_unix: 1729000001,
            },
          };
          const reply = {
            comment_id: replyCall,
            repository: 7,
            revision: 0,
            actor: 'builder',
            author_name: 'Builder',
            assignment: work,
            reply_to: okCall,
            body: replyTitle + '\n\n' + replySummary,
            at: 1729000004,
            communication: {
              kind: 'reply',
              title: replyTitle,
              summary: replySummary,
              reply_to: okCall,
            },
          };
          return Response.json({
            items: returned
              ? [
                  failed,
                  {
                    comment_id: okCall,
                    repository: 7,
                    revision: 0,
                    actor: 'builder',
                    author_name: 'Builder',
                    assignment: work,
                    reply_to: '',
                    body: 'Consulted JEV · choice',
                    at: 1729000002,
                    consultation: {
                      snapshot: 'b'.repeat(64),
                      group_id: 'd'.repeat(64),
                      assignment: work,
                      tool: 'systemone.choice',
                      kind: 'choice',
                      status: 'returned',
                      model: 'jev-recorded',
                      state: {amount: 1001, rule: 'Finance above EUR 1000'},
                      instructions: 'Does this request need Finance approval?',
                      criteria: {yes: 'Finance required', no: 'Manager only'},
                      created_unix: 1729000002,
                      updated_unix: 1729000003,
                      answer: {
                        answer: {
                          type: 'choice',
                          choice: 'yes',
                          probabilities: {yes: 0.96, no: 0.04},
                          confidence: 0.92,
                        },
                      },
                    },
                  },
                  reply,
                ]
              : [failed],
            next_before: 0,
          });
        }
        return null;
      });
    },
    {work, failCall, okCall, replyCall, replyTitle, replySummary}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByText('JEV request failed', {exact: true}).waitFor();
  await feed
    .getByText(
      'The recorded request failed. No result was recorded for this request. Refresh replies checks the stored outcome again.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await feed.getByText('nothing was decided', {exact: false}).count(), 0);
  await feed.getByText('Consulted JEV · choice', {exact: true}).waitFor();
  assert.equal(await feed.getByText(replySummary, {exact: false}).count(), 0);
  const draft = feed.locator(`textarea[data-feed-draft="record:${work}"]`);
  await draft.fill('Hold the EUR 420 request; do not submit yet.');
  const readsBefore = await page.evaluate(
    () => window.drawerFixture.calls.filter((call) => call.url.includes('/api/factory/feed/replies?')).length
  );
  await feed.getByRole('button', {name: 'Refresh replies', exact: true}).click();
  await feed.getByText('JEV returned a result', {exact: true}).waitFor();
  await feed.getByText('Selected: yes', {exact: false}).waitFor();
  await feed
    .locator(`#comment-${failCall}`)
    .getByText('No result was recorded for this request.', {exact: false})
    .waitFor();
  await feed.locator(`#comment-${failCall}`).getByText('Consulted JEV · choice', {exact: true}).waitFor();
  assert.equal(await feed.locator(`#comment-${failCall}`).getByText(replySummary, {exact: false}).count(), 0);
  await feed.locator(`#comment-${okCall}`).getByText('Selected: yes', {exact: false}).waitFor();
  const replyCard = feed.locator(`#comment-${replyCall}`);
  await replyCard.getByRole('heading', {name: replyTitle, exact: true}).waitFor();
  await replyCard.getByText(replySummary, {exact: false}).waitFor();
  assert.equal(await replyCard.locator('.soda-feed-body-text').count(), 0);
  await feed.getByText('JEV result', {exact: false}).waitFor();
  assert.equal(await draft.inputValue(), 'Hold the EUR 420 request; do not submit yet.');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
  const readsAfter = await page.evaluate(
    () => window.drawerFixture.calls.filter((call) => call.url.includes('/api/factory/feed/replies?')).length
  );
  assert.equal(readsAfter > readsBefore, true);
  // The drawer fixture replaces window.fetch with its API double (403 without
  // the fixture header), so static assets go through XHR, the same real
  // network path as the browser stylesheet loader.
  const get = (path: string) =>
    page.evaluate(
      (path) =>
        new Promise<{status: number; type: string; body: string}>((resolve, reject) => {
          const xhr = new XMLHttpRequest();
          xhr.open('GET', path);
          xhr.onload = () =>
            resolve({status: xhr.status, type: xhr.getResponseHeader('content-type') ?? '', body: xhr.responseText});
          xhr.onerror = () => reject(Error('asset request failed: ' + path));
          xhr.send();
        }),
      path
    );
  const conversation = await get('/assets/sodaspaces-factory-conversation.css');
  assert.equal(conversation.status, 200);
  assert.match(conversation.type ?? '', /text\/css/);
  assert.match(conversation.body, /\.soda-feed-filters/);
  const applied = await page.evaluate(() => {
    const feed = document.querySelector('soda-factory-feed') as unknown as
      | ({renderRoot?: ParentNode} & ParentNode)
      | null;
    const root = feed?.renderRoot ?? feed;
    const filters = root?.querySelector('.soda-feed-filters');
    const pre = root?.querySelector('.soda-feed-consultation pre');
    return {
      filtersDisplay: filters ? getComputedStyle(filters).display : 'missing',
      preWrap: pre ? getComputedStyle(pre as Element).whiteSpace : 'missing',
    };
  });
  assert.equal(applied.filtersDisplay, 'grid');
  assert.equal(applied.preWrap, 'pre-wrap');
  const icon = await get('/assets/soda/forgejo/icons/repo-19.14.0.svg');
  assert.equal(icon.status, 200);
  assert.match(icon.type ?? '', /svg/);
  assert.match(icon.body, /<svg/);
  for (const theme of ['light', 'dark'])
    for (const width of [1440, 800, 390]) {
      await page.setViewportSize({width, height: 900});
      await page.evaluate((value) => {
        document.documentElement.style.colorScheme = value;
      }, theme);
      await captureSpacesComponent(page, `feed-comments-${theme}-${width}`);
    }
});

test('an MCP question renders its content once and answer choices edit a reply without submitting it', async (t) => {
  const page = await fixture(t),
    postId = 'f'.repeat(32);
  await page.evaluate((postId) => {
    window.drawerFixture.api.setPresentation('settings');
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
          entries: [
            {
              kind: 'post',
              id: postId,
              repository: 7,
              seq: 42,
              actor: 'architect',
              author: 'Architect',
              project: 'soda-tester/demo',
              status: 'ask',
              activity: 'ask',
              payload: {
                communication: {
                  kind: 'ask',
                  component_id: 'S02',
                  title: 'Who can reassign tasks?',
                  summary: 'The reassignment permission changes the access plan.',
                  question: 'Should HR be allowed to reassign?',
                  options: ['Allow HR to reassign', 'Keep the original owner'],
                },
              },
            },
          ],
          truncated: false,
        });
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
  const bodyReads = await page.evaluate(
    (postId) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${postId}`)),
    postId
  );
  assert.equal(bodyReads.length, 0);
});

test('findings show supplied source literally and corrections retain before and after', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
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
              kind: 'post',
              id: 'c'.repeat(32),
              repository: 7,
              seq: 46,
              actor: 'test_pilot',
              author: 'Test Pilot',
              payload: {
                communication: {
                  kind: 'journey',
                  component_id: 'TP03',
                  title: 'Task journey observed',
                  summary: 'Recorded journey observation: passed. Application verification is unchanged.',
                  before: 'The task remains after refresh.',
                  after: 'The same task was visible after refresh.',
                  items: [{label: 'Refresh the task board', state: 'reported'}],
                },
              },
            },
            {
              kind: 'post',
              id: 'd'.repeat(32),
              repository: 7,
              seq: 45,
              actor: 'strategist',
              author: 'Strategist',
              payload: {
                communication: {
                  kind: 'diagnosis',
                  component_id: 'ST02',
                  title: 'Missing producer diagnosed',
                  summary: 'Recorded diagnosis (supported). Graph and readiness are unchanged.',
                  items: [
                    {
                      label: 'Missing fixture producer',
                      detail: 'The complete sealed inventory has no authorized producer. '.repeat(14),
                      state: 'reported_supported',
                    },
                  ],
                  alternatives: ['Renaming an issue does not supply the missing result.'],
                  remaining: ['The owner must independently accept the fixture.'],
                },
              },
            },
            {
              kind: 'post',
              id: 'e'.repeat(32),
              repository: 7,
              seq: 44,
              actor: 'strategist',
              author: 'Strategist',
              payload: {
                communication: {
                  kind: 'repair_draft',
                  component_id: 'ST06',
                  title: 'Fixture producer proposed',
                  summary:
                    'Recorded draft (repair). Not submitted for judgment or adopted; graph and readiness are unchanged.',
                  before: 'Current protected graph',
                  after: 'Proposed fixture producer',
                  items: [{id: 'producer', label: 'Add the missing fixture', state: 'proposed'}],
                  remaining: [
                    'Preview shows 1 of 1 proposed graph rows. The complete immutable report remains available.',
                  ],
                },
              },
            },
            {
              kind: 'post',
              id: 'a'.repeat(32),
              repository: 7,
              seq: 43,
              actor: 'fixer',
              author: 'Fixer',
              payload: {
                communication: {
                  kind: 'partial',
                  component_id: 'FX04',
                  title: 'Ownership corrected',
                  summary: 'Compare the person making the request.',
                  source: {
                    path: 'router.ts',
                    start_line: 12,
                    code: 'return !!task.ownerId;',
                    after: 'return task.ownerId === staff.id;',
                  },
                  commits: [{revision: 'b'.repeat(40), summary: 'Restrict access to owner'}],
                },
              },
            },
            {
              kind: 'post',
              id: 'b'.repeat(32),
              repository: 7,
              seq: 42,
              actor: 'inspector',
              author: 'Inspector',
              payload: {
                communication: {
                  kind: 'flag',
                  component_id: 'IN03',
                  title: 'Ownership defect',
                  summary: 'Another employee can open the task.',
                  source: {
                    path: 'router.ts',
                    start_line: 12,
                    code: '<script>window.untrustedSourceRan = true</script>\nreturn !!task.ownerId;',
                    highlight_lines: [13],
                  },
                  items: [{label: 'Other employee access', state: 'not passed'}],
                },
              },
            },
          ],
          truncated: false,
        });
      return null;
    });
  });
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Ownership defect'}).waitFor();
  assert.equal(await feed.locator('[data-action="finding"] .soda-action-source script').count(), 0);
  assert.match(
    await feed.locator('[data-action="finding"] .soda-action-source code').innerText(),
    /<script>window.untrustedSourceRan/
  );
  assert.equal(await feed.locator('[data-change="flagged"] > span').last().innerText(), 'return !!task.ownerId;');
  assert.equal(await feed.locator('[data-action="finding"] [data-tone="positive"]').count(), 0);
  assert.equal(await feed.locator('[data-action="correction"] pre').count(), 2);
  assert.equal(await feed.locator('.soda-action-commits code').innerText(), 'bbbbbbbb');
  assert.equal(await feed.locator('.soda-action-commits li > span').innerText(), 'Restrict access to owner');
  for (const [id, kind, action] of [
    ['c', 'journey', 'journey'],
    ['d', 'diagnosis', 'diagnosis'],
    ['e', 'repair_draft', 'comparison'],
  ] as const) {
    const card = feed.locator(`#feed-post-${id.repeat(32)}`);
    assert.equal(await card.locator(`[data-communication="${kind}"] [data-action="${action}"]`).count(), 1);
    assert.equal(await card.locator('[data-tone="positive"]').count(), 0);
  }
  assert.equal(await feed.getByText('Recorded detail unavailable', {exact: true}).count(), 0);
  assert.equal(await feed.getByText('Journey observation recorded', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Diagnosis recorded', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Repair proposed', {exact: true}).count(), 1);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('recorded suggestion and answer resolutions remain readable in the feed', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
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
              kind: 'post',
              id: 'a'.repeat(32),
              repository: 7,
              seq: 45,
              actor: 'architect',
              payload: {
                communication: {
                  kind: 'suggest',
                  title: 'Cache the policy',
                  summary: 'Avoid repeated reads.',
                  disposition: 'applied',
                  disposition_reason: 'Cached only within the assignment.',
                },
              },
            },
            {
              kind: 'post',
              id: 'b'.repeat(32),
              repository: 7,
              seq: 44,
              actor: 'architect',
              payload: {
                communication: {
                  kind: 'suggest',
                  title: 'Share the cache',
                  summary: 'Share reads between assignments.',
                  disposition: 'refused',
                  disposition_reason:
                    '<script>window.untrustedResolutionRan = true</script> would cross assignment scope.',
                },
              },
            },
            {
              kind: 'post',
              id: 'c'.repeat(32),
              repository: 7,
              seq: 43,
              actor: 'architect',
              payload: {
                communication: {
                  kind: 'ask',
                  title: 'Who can read tasks?',
                  summary: 'The access policy needs clarification.',
                  question_state: 'incorporated',
                },
              },
            },
            {
              kind: 'post',
              id: 'd'.repeat(32),
              repository: 7,
              seq: 42,
              actor: 'builder',
              payload: {
                communication: {
                  kind: 'answer',
                  title: 'Only the owner',
                  summary: 'The owner alone can read their tasks.',
                  disposition: 'incorporated',
                },
              },
            },
            {
              kind: 'post',
              id: 'e'.repeat(32),
              repository: 7,
              seq: 41,
              actor: 'architect',
              payload: {
                communication: {
                  kind: 'suggest',
                  title: 'Consider a bounded cache',
                  summary: 'Still awaiting a decision.',
                },
              },
            },
          ],
          truncated: false,
        });
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
  assert.equal(
    await feed.locator('[data-post-resolution="applied"]').innerText(),
    'Cached only within the assignment.'
  );
  assert.match(
    await feed.locator('[data-post-resolution="refused"]').innerText(),
    /<script>window.untrustedResolutionRan/
  );
  assert.equal(await feed.locator('[data-post-resolution] script').count(), 0);
});

test('every product change state renders its own outcome, action and tone', async (t) => {
  const page = await fixture(t),
    work = '9'.repeat(32),
    reviewable = 'a'.repeat(32),
    waiting = 'b'.repeat(32),
    drafted = 'c'.repeat(32),
    accepted = 'd'.repeat(32),
    superseded = 'e'.repeat(32),
    missing = 'f'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, reviewable, waiting, drafted, accepted, superseded, missing, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      document.body.addEventListener('soda-intent-open', (event) => {
        document.body.dataset.changeOpened = JSON.stringify((event as CustomEvent).detail);
      });
      const change = (state: string, can_review: boolean, draft_revision?: string) => ({
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        ...(draft_revision ? {draft_revision} : {}),
        state,
        can_review,
        verified: false,
        evidence_status: 'unavailable',
      });
      // Canonical Go wire states (feed_communication.go): a same-product review
      // starts at judgment applied, route new_issue, draft/effect pending,
      // issue 0 — scope-only rows with no native issue yet.
      const row = (comment_id: string, product_change: unknown) => ({
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        product_change,
      });
      // The accepted row is the owner (can_review) so the private request
      // control stays covered here; collaborator accepted routing is covered
      // by the role test below (evidence link, never the intent route).
      // The unavailable row is the explicit projection miss: the lane stays
      // product (accepted source, applied new_issue, no native issue) but
      // the detail did not arrive, so it must read as unavailable rather
      // than detached generic progress.
      const replies = [
        row(reviewable, change('review_required', true)),
        row(waiting, change('review_required', false)),
        row(drafted, change('draft', true, 'd' + 'd'.repeat(24))),
        row(accepted, change('accepted', true, 'd' + 'd'.repeat(24))),
        row(superseded, change('superseded', true)),
        row(missing, change('unavailable', false)),
      ];
      const byId: Record<string, unknown> = {};
      for (const item of replies) byId[(item as {comment_id: string}).comment_id] = item;
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
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) return Response.json({items: replies, next_before: 0});
        for (const id of Object.keys(byId))
          if (call.url.endsWith(`/api/factory/feed/comments/${id}`) && call.method !== 'POST')
            return Response.json(byId[id]);
        return null;
      });
    },
    {work, reviewable, waiting, drafted, accepted, superseded, missing, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  // Each change state owns outcome copy no other state shares.
  const outcomes: Array<[string, number]> = [
    ['Review this proposed change before updating the accepted direction.', 1],
    ['Waiting for the request owner to review this proposed change.', 1],
    ['The change is saved in the build request for review. Accept its direction before building.', 1],
    ['The changed direction is accepted. Follow its build and verification before applying a version.', 1],
    ['This proposal refers to an earlier direction. Refresh the request before proposing another change.', 1],
  ];
  for (const [text, count] of outcomes) assert.equal(await feed.getByText(text, {exact: true}).count(), count, text);
  // The unavailable miss offers a Check progress re-read in the same
  // outcome paragraph, so its copy asserts as a substring: it must read
  // as unavailable once, never as detached generic progress.
  assert.equal(
    await feed
      .getByText('The proposed change detail is currently unavailable. Check progress to reload it.', {
        exact: false,
      })
      .count(),
    1
  );
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 1);
  // Actions stay state-specific: review, continue and open each appear once;
  // the waiting and superseded rows offer no change button at all.
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Continue change review', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Open accepted change', exact: true}).count(), 1);
  assert.equal(await feed.locator('[data-comment-product-change]').count(), 3);
  // Tones stay distinct: only the superseded outcome raises an alert.
  assert.equal(
    await feed
      .getByText('The changed direction is accepted. Follow its build and verification before applying a version.', {
        exact: true,
      })
      .getAttribute('role'),
    'status'
  );
  assert.equal(
    await feed
      .getByText('This proposal refers to an earlier direction. Refresh the request before proposing another change.', {
        exact: true,
      })
      .getAttribute('role'),
    'alert'
  );
  // The pipeline stage list names each change state word for word: the
  // unavailable projection miss reads as its own product stage, never as
  // detached generic pipeline work.
  const stages = await feed.locator('.soda-feed-outcome-stages li').allTextContents();
  assert.deepEqual(
    stages
      .filter((line) => line.startsWith('Product change'))
      .map((line) => line.replace(/\s+/g, ' ').trim())
      .sort(),
    [
      'Product change — accepted',
      'Product change — draft',
      'Product change — review required',
      'Product change — review required',
      'Product change — superseded',
      'Product change — unavailable — current product detail unavailable',
    ].sort()
  );
  // Scope-only rows (issue 0, productChange present) render no native
  // issue-created/implemented/verified stage at all: the stage list stops
  // after the Product change word, never claiming native work.
  assert.deepEqual(
    stages
      .filter(
        (line) =>
          line.startsWith('Native issue') ||
          line.includes('implemented') ||
          line.includes('verified')
      )
      .map((line) => line.replace(/\s+/g, ' ').trim()),
    []
  );
  // Opening the accepted change dispatches the intent without any mutation.
  await feed.getByRole('button', {name: 'Open accepted change', exact: true}).click();
  await feed.getByText('Opening the accepted change.', {exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => JSON.parse(document.body.dataset.changeOpened ?? '{}')), {
    id: intent,
    repository: '7',
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('adopted change lineage renders evidence and native links separately from owner controls', async (t) => {
  const page = await fixture(t),
    work = '8'.repeat(32),
    verifiedRow = 'a'.repeat(32),
    pendingRow = 'b'.repeat(32),
    unknownRow = 'c'.repeat(32),
    historicRow = 'd'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, verifiedRow, pendingRow, unknownRow, historicRow, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      document.body.addEventListener('soda-intent-open', (event) => {
        document.body.dataset.changeOpened = JSON.stringify((event as CustomEvent).detail);
      });
      const adopted = 'd' + 'e'.repeat(24);
      const task = 'd' + 'f'.repeat(24);
      const acceptance = 'd' + '9'.repeat(24);
      const row = (comment_id: string, product_change: unknown) => ({
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        product_change,
      });
      const base = {
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        draft_revision: 'd' + 'd'.repeat(24),
        verified: false,
        evidence_status: 'unavailable',
      };
      const replies = [
        // Controller-verified adoption: a nonowner still opens the
        // collaborator-visible progress, while evidence and the native
        // destination render as separate facts.
        row(verifiedRow, {
          ...base,
          state: 'accepted',
          can_review: false,
          adopted_revision: adopted,
          scope_digest: '0'.repeat(64),
          verified: true,
          evidence_status: 'recorded',
          evidence: [{task_key: task, issue: 7, acceptance, completion: '1'.repeat(32)}],
        }),
        // Adopted but unverified: delivery recorded, no successful
        // application check recorded. The bound task carries its explicit
        // pending marker: the native binding is exact, but no completion
        // is inferred.
        row(pendingRow, {
          ...base,
          state: 'accepted',
          can_review: true,
          adopted_revision: adopted,
          scope_digest: '0'.repeat(64),
          evidence_status: 'recorded',
          evidence: [{task_key: task, issue: 7, acceptance, pending: true}],
        }),
        // No adoption yet: unknown work stays unknown, owner control kept.
        row(unknownRow, {...base, state: 'draft', can_review: true}),
        // Superseded head keeps its historical adoption and evidence.
        row(historicRow, {
          ...base,
          state: 'superseded',
          can_review: false,
          adopted_revision: adopted,
          scope_digest: '0'.repeat(64),
          evidence_status: 'recorded',
          evidence: [{task_key: task, issue: 7, acceptance, completion: '1'.repeat(32)}],
        }),
      ];
      const byId: Record<string, unknown> = {};
      for (const item of replies) byId[(item as {comment_id: string}).comment_id] = item;
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
                project: 'soda-tester/demo',
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) return Response.json({items: replies, next_before: 0});
        for (const id of Object.keys(byId))
          if (call.url.endsWith(`/api/factory/feed/comments/${id}`) && call.method !== 'POST')
            return Response.json(byId[id]);
        return null;
      });
    },
    {work, verifiedRow, pendingRow, unknownRow, historicRow, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  // Canonical evidence reads as text, native destinations as their own
  // links: verified, pending and historical rows each name one issue.
  await feed.getByText(/passed the approved application check/, {exact: false}).waitFor();
  // Native destinations live beside the evidence inside the processing
  // details: count them as links regardless of the collapsed section.
  const nativeLinks = feed.locator('a', {hasText: 'Open native issue #7'});
  assert.equal(await nativeLinks.count(), 3);
  assert.deepEqual(await nativeLinks.evaluateAll((links) => links.map((link) => link.getAttribute('href'))), [
    '/soda-tester/demo/issues/7',
    '/soda-tester/demo/issues/7',
    '/soda-tester/demo/issues/7',
  ]);
  await feed
    .getByText(/no successful approved application check recorded/, {exact: false})
    .first()
    .waitFor();
  const unknownEvidence =
    'Change work evidence is unavailable; no adopted delivery has been recorded for this proposal.';
  assert.equal(await feed.getByText(unknownEvidence, {exact: true}).count(), 1);
  // Owner-only controls stay gated while collaborator-visible progress
  // stays navigation-only: one Continue for the owner draft, one private
  // Open button for the owner accepted row, one evidence link for the
  // collaborator accepted row, no Review button.
  assert.equal(await feed.getByRole('button', {name: 'Continue change review', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Open accepted change', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 0);
  const collaboratorEvidence = feed.getByRole('link', {name: 'Open accepted evidence', exact: true});
  assert.equal(await collaboratorEvidence.count(), 1);
  assert.equal(await collaboratorEvidence.getAttribute('href'), '/soda-tester/demo/issues/7');
  // Lineage stages name the adopted change without claiming a native
  // issue creation.
  const stages = await feed.locator('.soda-feed-outcome-stages li').allTextContents();
  assert.equal(stages.filter((line) => line.replace(/\s+/g, ' ').trim().startsWith('Adopted change')).length, 3);
  assert.deepEqual(
    stages
      .filter((line) => line.trim().startsWith('Native issue') || line.includes('implemented'))
      .map((line) => line.replace(/\s+/g, ' ').trim()),
    []
  );
  // Opening the owner accepted row uses the private request control
  // without any mutation; the collaborator row never targets it.
  await feed.getByRole('button', {name: 'Open accepted change', exact: true}).click();
  await feed.getByText('Opening the accepted change.', {exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => JSON.parse(document.body.dataset.changeOpened ?? '{}')), {
    id: intent,
    repository: '7',
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('accepted change routes the owner to the private request and the collaborator to visible evidence', async (t) => {
  const page = await fixture(t),
    work = '7'.repeat(32),
    ownerRow = 'a'.repeat(32),
    collaboratorRow = 'b'.repeat(32),
    pendingRow = 'c'.repeat(32),
    unavailableRow = 'd'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, ownerRow, collaboratorRow, pendingRow, unavailableRow, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      document.body.addEventListener('soda-intent-open', (event) => {
        document.body.dataset.changeOpened = JSON.stringify((event as CustomEvent).detail);
        document.body.dataset.changeOpenCount = String(Number(document.body.dataset.changeOpenCount ?? '0') + 1);
      });
      const adopted = 'd' + 'e'.repeat(24);
      const task = 'd' + 'f'.repeat(24);
      const acceptance = 'd' + '9'.repeat(24);
      const base = {
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        draft_revision: 'd' + 'd'.repeat(24),
        verified: true,
        evidence_status: 'recorded',
      };
      const row = (comment_id: string, product_change: unknown) => ({
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        product_change,
      });
      const replies = [
        // Owner accepted: private request control with completed delivery.
        row(ownerRow, {
          ...base,
          state: 'accepted',
          can_review: true,
          adopted_revision: adopted,
          scope_digest: '0'.repeat(64),
          evidence: [{task_key: task, issue: 7, acceptance, completion: '1'.repeat(32)}],
        }),
        // Collaborator accepted: the same adopted delivery, visible only.
        row(collaboratorRow, {
          ...base,
          state: 'accepted',
          can_review: false,
          adopted_revision: adopted,
          scope_digest: '0'.repeat(64),
          evidence: [{task_key: task, issue: 7, acceptance, completion: '1'.repeat(32)}],
        }),
        // Collaborator accepted but pending: recorded graph, no completion
        // yet — counts as no delivered task, still links the native issue.
        row(pendingRow, {
          ...base,
          state: 'accepted',
          can_review: false,
          verified: false,
          adopted_revision: adopted,
          scope_digest: '0'.repeat(64),
          evidence: [{task_key: task, issue: 7, acceptance}],
        }),
        // Collaborator accepted with unknown task evidence: known adoption
        // and verification still read, without asserting an absence.
        row(unavailableRow, {
          ...base,
          state: 'accepted',
          can_review: false,
          evidence_status: 'unavailable',
          adopted_revision: adopted,
          scope_digest: '0'.repeat(64),
        }),
      ];
      const byId: Record<string, unknown> = {};
      for (const item of replies) byId[(item as {comment_id: string}).comment_id] = item;
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
                project: 'soda-tester/demo',
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) return Response.json({items: replies, next_before: 0});
        for (const id of Object.keys(byId))
          if (call.url.endsWith(`/api/factory/feed/comments/${id}`) && call.method !== 'POST')
            return Response.json(byId[id]);
        return null;
      });
    },
    {work, ownerRow, collaboratorRow, pendingRow, unavailableRow, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByText(/passed the approved application check/, {exact: false}).first().waitFor();
  // Owner keeps the single private request control; collaborators never do.
  assert.equal(await feed.getByRole('button', {name: 'Open accepted change', exact: true}).count(), 1);
  assert.equal(await feed.locator('[data-comment-product-change]').count(), 1);
  // Collaborator-visible evidence: two recorded rows link the native issue
  // in details plus two primary evidence links; the unavailable row stays
  // link-free while still naming its adopted revision.
  const evidenceLinks = feed.getByRole('link', {name: 'Open accepted evidence', exact: true});
  assert.equal(await evidenceLinks.count(), 2);
  for (const href of await evidenceLinks.evaluateAll((links) => links.map((link) => link.getAttribute('href'))))
    assert.equal(href, '/soda-tester/demo/issues/7');
  const nativeLinks = feed.locator('a', {hasText: 'Open native issue #7'});
  assert.equal(await nativeLinks.count(), 3);
  // Pending delivery counts zero completed tasks instead of one delivered.
  await feed
    .getByText(/no delivered task covers this change yet/, {exact: false})
    .first()
    .waitFor();
  assert.equal(await feed.getByText(/1 delivered task recorded/, {exact: false}).count(), 2);
  // Unavailable task evidence names the known adoption and verification
  // without asserting that no delivery exists.
  await feed
    .getByText(/task delivery unknown/, {exact: false})
    .first()
    .waitFor();
  assert.equal(await feed.getByText(/no adopted delivery has been recorded/, {exact: false}).count(), 0);
  const stages = await feed.locator('.soda-feed-outcome-stages li').allTextContents();
  assert.equal(stages.filter((line) => line.replace(/\s+/g, ' ').trim().startsWith('Adopted change')).length, 4);
  assert.ok(stages.some((line) => line.replace(/\s+/g, ' ').trim().startsWith('Change task evidence')));
  // Owner click lands on the private intent route without any mutation.
  await feed.getByRole('button', {name: 'Open accepted change', exact: true}).click();
  await page.waitForFunction(() => document.body.dataset.changeOpenCount === '1');
  assert.deepEqual(await page.evaluate(() => JSON.parse(document.body.dataset.changeOpened ?? '{}')), {
    id: intent,
    repository: '7',
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('consultation request and interpretation reply each carry the consulting worker attribution', async (t) => {
  const page = await fixture(t),
    work = '1'.repeat(32),
    call = '2'.repeat(32),
    reply = '3'.repeat(32),
    // Same declared follow-up fixture (live_test.go:849 shape): a separate
    // factory.reply with reply_to set, never consultation body text. This
    // text is authored test content, not an actual production/provider
    // outcome. The test asserts chronological request-then-interpretation,
    // not explanation placement above the block.
    replyTitle = 'Approval research recorded',
    replySummary =
      'I retained all three actual judgments with the supplied policy. My next action is to cite the policy in the approval design review; this research does not approve the request or implement a change.';
  await page.evaluate(
    ({work, call, reply, replyTitle, replySummary}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (request) => {
        if (request.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (request.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (request.url.includes('/api/factory/feed?'))
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
                summary: 'Working on the equipment request app.',
              },
            ],
            truncated: false,
          });
        if (request.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items: [
              {
                comment_id: call,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: '',
                body: 'Consulted JEV · choice',
                at: 1729000000,
                consultation: {
                  snapshot: 'a'.repeat(64),
                  group_id: 'b'.repeat(64),
                  assignment: work,
                  tool: 'systemone.choice',
                  kind: 'choice',
                  status: 'returned',
                  model: 'jev-recorded',
                  state: {amount: 1001, rule: 'Finance above EUR 1000'},
                  instructions: 'Does this request need Finance approval?',
                  criteria: {yes: 'Finance required', no: 'Manager only'},
                  created_unix: 1729000000,
                  updated_unix: 1729000001,
                  answer: {
                    answer: {
                      type: 'choice',
                      choice: 'yes',
                      probabilities: {yes: 0.96, no: 0.04},
                      confidence: 0.92,
                    },
                  },
                },
              },
              {
                comment_id: reply,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: call,
                body: replyTitle + '\n\n' + replySummary,
                at: 1729000002,
                communication: {
                  kind: 'reply',
                  title: replyTitle,
                  summary: replySummary,
                  reply_to: call,
                },
              },
            ],
            next_before: 0,
          });
        return null;
      });
    },
    {work, call, reply, replyTitle, replySummary}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const requestCard = feed.locator(`#comment-${call}`);
  await requestCard.getByText('Consulted JEV · choice', {exact: true}).waitFor();
  assert.equal(await requestCard.locator('.soda-feed-comment-who strong').innerText(), 'Builder');
  assert.equal(await requestCard.getAttribute('data-role'), 'builder');
  const replyCard = feed.locator(`#comment-${reply}`);
  await replyCard.getByRole('heading', {name: replyTitle, exact: true}).waitFor();
  assert.equal(await replyCard.locator('.soda-feed-comment-who strong').innerText(), 'Builder');
  assert.equal(await replyCard.getAttribute('data-role'), 'builder');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('consultation interpretation renders after its request in the work thread', async (t) => {
  const page = await fixture(t),
    work = '4'.repeat(32),
    call = '5'.repeat(32),
    reply = '6'.repeat(32),
    replyTitle = 'Approval research recorded',
    replySummary =
      'I retained all three actual judgments with the supplied policy. My next action is to cite the policy in the approval design review; this research does not approve the request or implement a change.';
  await page.evaluate(
    ({work, call, reply, replyTitle, replySummary}) => {
      let reads = 0;
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (request) => {
        if (request.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (request.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (request.url.includes('/api/factory/feed?'))
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
                summary: 'Working on the equipment request app.',
              },
            ],
            truncated: false,
          });
        if (request.url.includes('/api/factory/feed/replies?')) {
          const answered = reads++ > 0;
          const consultation = {
            comment_id: call,
            repository: 7,
            revision: 0,
            actor: 'builder',
            author_name: 'Builder',
            assignment: work,
            reply_to: '',
            body: 'Consulted JEV · choice',
            at: 1729000000,
            consultation: {
              snapshot: 'c'.repeat(64),
              group_id: 'd'.repeat(64),
              assignment: work,
              tool: 'systemone.choice',
              kind: 'choice',
              status: answered ? 'returned' : 'pending',
              state: {amount: 1001, rule: 'Finance above EUR 1000'},
              instructions: 'Does this request need Finance approval?',
              criteria: {yes: 'Finance required', no: 'Manager only'},
              created_unix: 1729000000,
              updated_unix: 1729000001,
              ...(answered
                ? {
                    model: 'jev-recorded',
                    answer: {
                      answer: {
                        type: 'choice',
                        choice: 'yes',
                        probabilities: {yes: 0.96, no: 0.04},
                        confidence: 0.92,
                      },
                    },
                  }
                : {}),
            },
          };
          // The thread API serves pages newest-first (seq DESC); the reader
          // reverses each page into ascending canonical order before
          // display, so the fixture serves the reply before the request.
          const interpretation = {
            comment_id: reply,
            repository: 7,
            revision: 0,
            actor: 'builder',
            author_name: 'Builder',
            assignment: work,
            reply_to: call,
            body: replyTitle + '\n\n' + replySummary,
            at: 1729000002,
            communication: {
              kind: 'reply',
              title: replyTitle,
              summary: replySummary,
              reply_to: call,
            },
          };
          return Response.json({
            items: answered ? [interpretation, consultation] : [consultation],
            next_before: 0,
          });
        }
        return null;
      });
    },
    {work, call, reply, replyTitle, replySummary}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByText('Waiting for JEV', {exact: true}).waitFor();
  assert.equal(await feed.locator(`#comment-${reply}`).count(), 0);
  await feed.getByRole('button', {name: 'Refresh replies', exact: true}).click();
  await feed.locator(`#comment-${reply}`).getByRole('heading', {name: replyTitle, exact: true}).waitFor();
  const ids = await feed.locator('li.soda-feed-comment').evaluateAll((nodes) => nodes.map((node) => node.id));
  assert.ok(ids.indexOf(`comment-${call}`) !== -1 && ids.indexOf(`comment-${reply}`) !== -1);
  assert.ok(ids.indexOf(`comment-${call}`) < ids.indexOf(`comment-${reply}`));
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('consultation request and interpretation reply stay visible with the explanation on the reply', async (t) => {
  const page = await fixture(t),
    work = '8'.repeat(32),
    call = '9'.repeat(32),
    reply = 'a'.repeat(32),
    replyTitle = 'Approval research recorded',
    replySummary =
      'I retained all three actual judgments with the supplied policy. My next action is to cite the policy in the approval design review; this research does not approve the request or implement a change.';
  await page.evaluate(
    ({work, call, reply, replyTitle, replySummary}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (request) => {
        if (request.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (request.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (request.url.includes('/api/factory/feed?'))
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
                summary: 'Working on the equipment request app.',
              },
            ],
            truncated: false,
          });
        if (request.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items: [
              {
                comment_id: call,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: '',
                body: 'Consulted JEV · choice',
                at: 1729000000,
                consultation: {
                  snapshot: 'e'.repeat(64),
                  group_id: 'f'.repeat(64),
                  assignment: work,
                  tool: 'systemone.choice',
                  kind: 'choice',
                  status: 'returned',
                  model: 'jev-recorded',
                  state: {amount: 1001, rule: 'Finance above EUR 1000'},
                  instructions: 'Does this request need Finance approval?',
                  criteria: {yes: 'Finance required', no: 'Manager only'},
                  created_unix: 1729000000,
                  updated_unix: 1729000001,
                  answer: {
                    answer: {
                      type: 'choice',
                      choice: 'yes',
                      probabilities: {yes: 0.96, no: 0.04},
                      confidence: 0.92,
                    },
                  },
                },
              },
              {
                comment_id: reply,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: call,
                body: replyTitle + '\n\n' + replySummary,
                at: 1729000002,
                communication: {
                  kind: 'reply',
                  title: replyTitle,
                  summary: replySummary,
                  reply_to: call,
                },
              },
            ],
            next_before: 0,
          });
        return null;
      });
    },
    {work, call, reply, replyTitle, replySummary}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const requestCard = feed.locator(`#comment-${call}`);
  const replyCard = feed.locator(`#comment-${reply}`);
  await requestCard.getByText('Consulted JEV · choice', {exact: true}).waitFor();
  await replyCard.getByRole('heading', {name: replyTitle, exact: true}).waitFor();
  assert.equal(await requestCard.isVisible(), true);
  assert.equal(await replyCard.isVisible(), true);
  // Explanation placement matches production: the interpretation lives on
  // the reply record, never inside the consultation record body.
  await replyCard.getByText(replySummary, {exact: false}).waitFor();
  assert.equal(await requestCard.getByText(replySummary, {exact: false}).count(), 0);
  assert.equal(await replyCard.getByText('Consulted JEV · choice', {exact: true}).count(), 0);
  assert.equal(await feed.getByText('Recorded detail unavailable', {exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('an unavailable change review reports honestly without opening or saving', async (t) => {
  const page = await fixture(t),
    work = 'e'.repeat(32),
    comment = 'a'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, comment, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      document.body.addEventListener('soda-intent-open', () => {
        document.body.dataset.changeOpenCount = String(Number(document.body.dataset.changeOpenCount ?? '0') + 1);
      });
      const base = {
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        verified: false,
        evidence_status: 'unavailable',
      };
      const reviewable = () => ({
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        product_change: {...base, state: 'review_required', can_review: true},
      });
      // The fresh change read misses the lane detail: an explicit
      // unavailable projection, never a save and never an absence claim.
      const unavailable = () => ({...reviewable(), product_change: {...base, state: 'unavailable', can_review: false}});
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
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({items: [reviewable()], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method !== 'POST')
          return Response.json(unavailable());
        return null;
      });
    },
    {work, comment, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  // The prior visible review button starts the transition.
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
  // Truthful message: unavailable and retryable, never a save claim.
  await feed
    .getByText('The proposed change detail is currently unavailable. Check progress to reload it.', {exact: true})
    .waitFor();
  assert.equal(
    await feed
      .getByText('Change saved for review. Review and accept the direction in the same build request.', {
        exact: true,
      })
      .count(),
    0
  );
  // No navigation and no save: zero intent events, zero mutations.
  assert.equal(await page.evaluate(() => Number(document.body.dataset.changeOpenCount ?? '0')), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
  const reads = await page.evaluate(
    (comment) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${comment}`)).length,
    comment
  );
  assert.equal(reads, 1);
  // Busy is released: refreshing the replies restores the visible review
  // button, and a second review click issues a fresh change GET instead of
  // sticking behind the finished review. A stuck busy flag would return
  // early here with no second read.
  await feed.getByRole('button', {name: 'Refresh replies', exact: true}).click();
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
  await page.waitForFunction(
    (expected) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${expected}`)).length >
      1,
    comment
  );
  await feed
    .getByText('The proposed change detail is currently unavailable. Check progress to reload it.', {exact: true})
    .waitFor();
  assert.equal(await page.evaluate(() => Number(document.body.dataset.changeOpenCount ?? '0')), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a nested unavailable reply recovers its known change through Check progress with reads only', async (t) => {
  const page = await fixture(t),
    work = 'e'.repeat(32),
    comment = 'b'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, comment, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      document.body.addEventListener('soda-intent-open', () => {
        document.body.dataset.changeOpenCount = String(Number(document.body.dataset.changeOpenCount ?? '0') + 1);
      });
      const base = {
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        verified: false,
        evidence_status: 'unavailable',
      };
      const row = (product_change: unknown) => ({
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        product_change,
      });
      // The drawer thread serves the lane without its detail; the direct
      // comment GET recovers the known review state. Reads only, no POST.
      const unavailable = row({...base, state: 'unavailable', can_review: false});
      const known = row({...base, state: 'review_required', can_review: true});
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
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({items: [unavailable], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method !== 'POST')
          return Response.json(known);
        return null;
      });
    },
    {work, comment, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  // The nested drawer row starts unavailable with its own Check progress;
  // view.outcomes stays empty in this reply flow, so only a thread-aware
  // lookup can reach it.
  const nested = feed.locator(`#comment-${comment}`);
  await nested
    .getByText('The proposed change detail is currently unavailable. Check progress to reload it.', {exact: false})
    .waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 0);
  const readsBefore = await page.evaluate(
    (comment) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${comment}`)).length,
    comment
  );
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  // The same visible thread row recovers in place: review copy plus its
  // own Review button, from exactly one comment GET.
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await nested
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor();
  const readsAfter = await page.evaluate(
    (comment) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${comment}`)).length,
    comment
  );
  assert.equal(readsAfter, readsBefore + 1);
  // GET-only recovery: no mutation and no navigation into the intent route.
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
  assert.equal(await page.evaluate(() => Number(document.body.dataset.changeOpenCount ?? '0')), 0);
  // Single surface: the recovered control lives in the nested row only —
  // the parent drawer never duplicates the same outcome above the thread.
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
});

test('a delayed Check progress GET cannot overwrite a newer Refresh replies state', async (t) => {
  const page = await fixture(t),
    work = 'e'.repeat(32),
    comment = 'b'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, comment, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      const base = {
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        verified: false,
        evidence_status: 'unavailable',
      };
      const row = (product_change: unknown, revision: number) => ({
        comment_id: comment,
        repository: 7,
        revision,
        intake: 'recorded',
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        product_change,
      });
      // The first thread page misses the lane detail. The refresh serves
      // the newer revision with review permission; the gated comment GET
      // resolves afterwards with the older unavailable projection.
      const stale = row({...base, state: 'unavailable', can_review: false}, 0);
      const newer = row({...base, state: 'review_required', can_review: true}, 1);
      let replies = 0;
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
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          replies++;
          return Response.json({items: [replies === 1 ? stale : newer], next_before: 0});
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method !== 'POST') {
          // A slow retry: stay pending until the test refreshes past it.
          while (document.body.dataset.staleReleased !== '1') await new Promise((run) => setTimeout(run, 25));
          return Response.json(stale);
        }
        return null;
      });
    },
    {work, comment, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  // Start the slow retry GET, then refresh past it: the refresh lands
  // first with the newer revision and review permission.
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  await page.waitForFunction(
    (expected) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${expected}`)).length >
      0,
    comment
  );
  await feed.getByRole('button', {name: 'Refresh replies', exact: true}).click();
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  // The delayed stale GET lands now; the newer thread state must survive
  // it, and the retry must release so later checks stay available.
  await page.evaluate(() => {
    document.body.dataset.staleReleased = '1';
  });
  await page.waitForFunction((id) => {
    const row = document.querySelector(`#comment-${id}`);
    if (!row) return false;
    const review = Array.from(row.querySelectorAll('button')).find((candidate) =>
      candidate.textContent?.includes('Review proposed change')
    );
    if (review instanceof HTMLButtonElement && !review.disabled) return true;
    return row.textContent?.includes('currently unavailable') ?? false;
  }, comment);
  // Short window: the 10s feed auto-refresh would eventually heal a
  // stale overwrite, so the newer state must already be intact here.
  await nested
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor({timeout: 3000});
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(
    await feed
      .getByText('The proposed change detail is currently unavailable. Check progress to reload it.', {exact: false})
      .count(),
    0
  );
  assert.equal(await nested.getByRole('button', {name: 'Review proposed change', exact: true}).isEnabled(), true);
  // Reads only: no mutation while racing the retry against the refresh.
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a recovered change detail clears the unavailable banner through Check progress', async (t) => {
  const page = await fixture(t),
    work = 'e'.repeat(32),
    comment = 'a'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, comment, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      document.body.addEventListener('soda-intent-open', () => {
        document.body.dataset.changeOpenCount = String(Number(document.body.dataset.changeOpenCount ?? '0') + 1);
      });
      const base = {
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        verified: false,
        evidence_status: 'unavailable',
      };
      const row = (product_change: unknown) => ({
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        product_change,
      });
      // The review read misses the lane detail; the later Check progress
      // re-read recovers the known review state. Reads only, no POST.
      const unavailable = row({...base, state: 'unavailable', can_review: false});
      const known = row({...base, state: 'review_required', can_review: true});
      let commentReads = 0;
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
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) return Response.json({items: [known], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method !== 'POST') {
          commentReads++;
          return Response.json(commentReads === 1 ? unavailable : known);
        }
        return null;
      });
    },
    {work, comment, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  const banner = feed.locator('[data-feed-message]');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  // The stale review reports the miss honestly without opening or saving.
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
  await banner
    .filter({hasText: 'The proposed change detail is currently unavailable. Check progress to reload it.'})
    .waitFor();
  assert.equal(await page.evaluate(() => Number(document.body.dataset.changeOpenCount ?? '0')), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
  // Check progress recovers the same row and retires the banner: exactly
  // one comment GET, still no mutation and no navigation.
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await nested
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor();
  assert.equal(
    await banner
      .filter({hasText: 'The proposed change detail is currently unavailable. Check progress to reload it.'})
      .count(),
    0
  );
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(
    await page.evaluate(
      (expected) =>
        window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${expected}`)).length,
      comment
    ),
    2
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
  assert.equal(await page.evaluate(() => Number(document.body.dataset.changeOpenCount ?? '0')), 0);
});

test('a newer message survives the same unavailable recovery', async (t) => {
  const page = await fixture(t),
    work = 'e'.repeat(32),
    comment = 'a'.repeat(32),
    posted = 'c'.repeat(32),
    intent = 'd' + 'b'.repeat(24);
  await page.evaluate(
    ({work, comment, posted, intent}) => {
      window.drawerFixture.api.setPresentation('settings');
      const base = {
        intent,
        accepted_revision: 'd' + 'c'.repeat(24),
        verified: false,
        evidence_status: 'unavailable',
      };
      const row = (comment_id: string, product_change?: unknown) => ({
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        body: 'Please review the EUR 420 equipment request change.',
        author_name: 'Maya',
        at: 1729000000,
        ...(product_change === undefined ? {} : {product_change}),
      });
      // The first page offers the review; later pages hold the miss until
      // the recovery re-read lands, so a submit in between cannot hide the
      // pending unavailable state.
      const unavailable = row(comment, {...base, state: 'unavailable', can_review: false});
      const known = {...row(comment, {...base, state: 'review_required', can_review: true}), route: 'new_issue'};
      let commentReads = 0,
        replyReads = 0;
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
                summary: 'Equipment request version 1',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          replyReads++;
          // The submit's readLatest fans out one cached-thread re-read of
          // this row before the replies re-read lands, so the manual Check
          // progress recovery below is the third comment read, not the
          // second. The row must still read unavailable until that Check.
          return Response.json({items: [replyReads === 1 || commentReads >= 3 ? known : unavailable], next_before: 0});
        }
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST')
          return Response.json(row(posted));
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method !== 'POST') {
          commentReads++;
          return Response.json(commentReads <= 2 ? unavailable : known);
        }
        if (call.url.includes('/api/factory/feed/comments/') && call.method !== 'POST')
          return Response.json(row(posted));
        return null;
      });
    },
    {work, comment, posted, intent}
  );
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  const banner = feed.locator('[data-feed-message]');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
  await banner
    .filter({hasText: 'The proposed change detail is currently unavailable. Check progress to reload it.'})
    .waitFor();
  // A newer submit lands before the recovery re-read; its message owns the
  // banner now.
  await feed.locator(`textarea[data-feed-draft="record:${work}"]`).fill('A newer human note.');
  await feed.locator(`button[data-comment-submit="record:${work}"]`).click();
  await banner.filter({hasText: 'Comment saved.'}).waitFor();
  // The recovery still heals the nested row but must leave the newer
  // message alone.
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  assert.equal(await banner.filter({hasText: 'Comment saved.'}).count(), 1);
  assert.equal(
    await banner
      .filter({hasText: 'The proposed change detail is currently unavailable. Check progress to reload it.'})
      .count(),
    0
  );
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
});

import test from 'node:test';
import assert from 'node:assert/strict';
import {
  commentPostBody,
  commentSourceForEntry,
  commentUnsettled,
  feedPageMaxEntries,
  feedQueryString,
  mergeFeedEntries,
  newIdempotencyKey,
  parseCommentDetail,
  parseCommentPipelineState,
  parseFeedCursor,
  parseFeedEntry,
  parseFeedPageView,
  parseFeedPayload,
  parseFeedStatusSegments,
  parseFeedSubscriptionView,
} from '../../frontend/spaces/sodaspaces-factory-feed-response.js';
import type {FeedEntry} from '../../frontend/spaces/sodaspaces-factory-feed-response.js';
import {
  describeCommentOutcome,
  describeDecisionAttention,
  describeDiscussion,
  describeFeedPost,
  milestoneText,
  describePlanIssue,
  describePostBody,
  describePostHeader,
  describeWorkEvidence,
  repositoryRef,
} from '../../frontend/spaces/sodaspaces-factory-feed-view.js';
import {
  classifyFeedStatus,
  describeFeedFamily,
  roleLabel,
  selectFeedFamily,
} from '../../frontend/spaces/sodaspaces-factory-feed-families.js';

const record: FeedEntry = {kind: 'record', id: 'a'.repeat(32), repository: 7, seq: 41};
const post: FeedEntry = {kind: 'post', id: 'comment-9', repository: 9, seq: 42};
const repo = repositoryRef(7, '/soda-tester/demo', 'soda-tester', 'demo');

test('feed entries admit the released kinds with positive scope and order', () => {
  assert.deepEqual(parseFeedEntry({kind: 'record', id: 'a'.repeat(32), repository: 7, seq: 41}), record);
  assert.deepEqual(parseFeedEntry({kind: 'post', id: 'comment-9', repository: 9, seq: 42}), post);
  for (const bad of [
    {kind: 'tweet', id: 'x', repository: 7, seq: 1},
    {kind: 'record', id: '', repository: 7, seq: 1},
    {kind: 'record', id: 'x', repository: 0, seq: 1},
    {kind: 'record', id: 'x', repository: 7, seq: -3},
    {kind: 'record', id: 'x', repository: 7.5, seq: 1},
    {kind: 'record', id: 'x'.repeat(257), repository: 7, seq: 1},
  ])
    assert.throws(() => parseFeedEntry(bad));
});

test('feed cursors stay opaque versioned order bounds', () => {
  assert.equal(parseFeedCursor(''), '');
  assert.equal(parseFeedCursor('v1:41'), 'v1:41');
  for (const bad of ['v9:1', 'v1:0', 'v1:-3', 'v1:1:2', 'latest', 41, 'v1:1.5'])
    assert.throws(() => parseFeedCursor(bad));
});

test('feed pages keep entries, resume cursor and truncation', () => {
  const page = parseFeedPageView({
    entries: [
      {kind: 'record', id: 'a'.repeat(32), repository: 7, seq: 42},
      {kind: 'record', id: 'b'.repeat(32), repository: 9, seq: 41},
    ],
    next: 'v1:41',
    truncated: true,
  });
  assert.equal(page.entries.length, 2);
  assert.equal(page.next, 'v1:41');
  assert.equal(page.truncated, true);
  const tail = parseFeedPageView({entries: [], truncated: false});
  assert.equal(tail.next, '');
  assert.throws(() => parseFeedPageView({entries: [], truncated: false, next: 'v9:1'}));
  assert.throws(() => parseFeedPageView({entries: 'none', next: '', truncated: false}));
});

test('feed query strings match the released route shape', () => {
  const query = feedQueryString({repositories: [9, 7, 7], limit: 20, kinds: [], cursor: ''});
  assert.match(query, /^factory\/feed\?/);
  const params = new URLSearchParams(query.slice('factory/feed?'.length));
  assert.equal(params.get('repositories'), '7,9');
  assert.equal(params.get('limit'), '20');
  assert.equal(params.get('kinds'), null);
  assert.equal(params.get('cursor'), null);
  const resumed = new URLSearchParams(
    feedQueryString({repositories: [7], limit: 20, kinds: ['record'], cursor: 'v1:41'}).slice('factory/feed?'.length)
  );
  assert.equal(resumed.get('kinds'), 'record');
  assert.equal(resumed.get('cursor'), 'v1:41');
  assert.throws(() => feedQueryString({repositories: [], limit: 20, kinds: [], cursor: ''}));
  assert.throws(() => feedQueryString({repositories: [7], limit: 257, kinds: [], cursor: ''}));
});

test('feed subscriptions admit saved scope and fresh empty scope', () => {
  assert.deepEqual(
    parseFeedSubscriptionView({subscriber: 'reader-1', repositories: [7, 9], kinds: ['record'], revision: 2}),
    {
      subscriber: 'reader-1',
      repositories: [7, 9],
      kinds: ['record'],
      revision: 2,
    }
  );
  const fresh = parseFeedSubscriptionView({subscriber: 'reader-1', repositories: null, kinds: null, revision: 0});
  assert.deepEqual(fresh.repositories, []);
  assert.deepEqual(fresh.kinds, []);
  for (const bad of [
    {subscriber: '', repositories: [7], revision: 1},
    {subscriber: 'reader-1', repositories: [], revision: 1},
    {subscriber: 'reader-1', repositories: [0], revision: 1},
    {subscriber: 'reader-1', repositories: [7], kinds: ['tweet'], revision: 1},
    {subscriber: 'reader-1', repositories: [7], revision: -1},
  ])
    assert.throws(() => parseFeedSubscriptionView(bad));
});

test('merging keeps newest-first order with stable identities', () => {
  const older: FeedEntry = {kind: 'record', id: 'old', repository: 7, seq: 40};
  const merged = mergeFeedEntries([record], [older, record, post]);
  assert.deepEqual(
    merged.map((entry) => entry.seq),
    [42, 41, 40]
  );
  assert.equal(merged.filter((entry) => entry.id === record.id).length, 1);
});

test('refreshing replaces an assignment result without duplicating its post', () => {
  const running: FeedEntry = {kind: 'record', id: 'run', repository: 7, seq: 40, status: 'implement:assigned'};
  const finished: FeedEntry = {...running, status: 'implement:finished:succeeded', summary: 'Checklist saved'};
  const entries = mergeFeedEntries([running], [finished]);
  assert.deepEqual(entries, [finished]);
});

test('feed entries admit optional server-derived enrichment', () => {
  const enriched = parseFeedEntry({
    kind: 'post',
    id: 'c'.repeat(32),
    repository: 9,
    seq: 42,
    actor: 'Product Lead',
    author: 'Ada',
    project: 'soda-tester/demo',
    at: 1720000000,
    status: 'assessing',
  });
  assert.equal(enriched.actor, 'Product Lead');
  assert.equal(enriched.author, 'Ada');
  assert.equal(enriched.project, 'soda-tester/demo');
  assert.equal(enriched.at, 1720000000);
  assert.equal(enriched.status, 'assessing');
  assert.deepEqual(parseFeedEntry({kind: 'record', id: 'x', repository: 7, seq: 1}), {
    kind: 'record',
    id: 'x',
    repository: 7,
    seq: 1,
  });
  for (const bad of [
    {kind: 'post', id: 'c'.repeat(32), repository: 9, seq: 42, actor: ''},
    {kind: 'post', id: 'c'.repeat(32), repository: 9, seq: 42, at: -1},
    {kind: 'post', id: 'c'.repeat(32), repository: 9, seq: 42, status: 'x'.repeat(65)},
  ])
    assert.throws(() => parseFeedEntry(bad));
});

test('feed entries admit additive recorded content without fabricating it', () => {
  const candidate = 'c'.repeat(40);
  const enriched = parseFeedEntry({
    kind: 'record',
    id: 'b'.repeat(32),
    repository: 7,
    seq: 50,
    issue: 3,
    pr: 42,
    ref: candidate,
    summary: 'built the candidate',
  });
  assert.equal(enriched.issue, 3);
  assert.equal(enriched.pr, 42);
  assert.equal(enriched.ref, candidate);
  assert.equal(enriched.summary, 'built the candidate');
  const bare = parseFeedEntry({kind: 'record', id: 'x', repository: 7, seq: 1});
  assert.equal(bare.issue, undefined);
  assert.equal(bare.pr, undefined);
  assert.equal(bare.ref, undefined);
  assert.equal(bare.summary, undefined);
  assert.equal(parseFeedEntry({kind: 'record', id: 'x', repository: 7, seq: 1, issue: 9}).issue, 9);
  // The summary bound counts runes, so 512 astral characters (1024
  // UTF-16 code units) still admit exactly as served.
  assert.equal(
    parseFeedEntry({kind: 'record', id: 'x', repository: 7, seq: 1, summary: '😀'.repeat(512)}).summary,
    '😀'.repeat(512)
  );
  for (const bad of [
    {kind: 'record', id: 'x', repository: 7, seq: 1, issue: 0},
    {kind: 'record', id: 'x', repository: 7, seq: 1, issue: -1},
    {kind: 'record', id: 'x', repository: 7, seq: 1, issue: '3'},
    {kind: 'record', id: 'x', repository: 7, seq: 1, pr: 1.5},
    {kind: 'record', id: 'x', repository: 7, seq: 1, ref: ''},
    {kind: 'record', id: 'x', repository: 7, seq: 1, ref: 'r'.repeat(257)},
    {kind: 'record', id: 'x', repository: 7, seq: 1, summary: ''},
    {kind: 'record', id: 'x', repository: 7, seq: 1, summary: 'é'.repeat(513)},
    {kind: 'record', id: 'x', repository: 7, seq: 1, summary: 42},
  ])
    assert.throws(() => parseFeedEntry(bad));
});

test('post headers keep author context outside the card without inventing facts', () => {
  const header = describePostHeader(record, repo);
  assert.equal(header.project, 'soda-tester/demo');
  assert.match(header.timeText, /#41/);
  assert.equal(header.role, 'factory record');
  const unknown = describePostHeader(record, repositoryRef(11, undefined, undefined, undefined));
  assert.equal(unknown.project, 'repository 11');
  const enriched = describePostHeader(
    {
      kind: 'post',
      id: 'c'.repeat(32),
      repository: 9,
      seq: 42,
      actor: 'Product Lead',
      author: 'Ada',
      project: 'soda-tester/demo',
      at: 1720000000,
      status: 'assessing',
    },
    repositoryRef(9, '/other/repo', 'other', 'repo')
  );
  assert.equal(enriched.name, 'Ada');
  assert.equal(enriched.role, 'Product Lead');
  assert.equal(enriched.project, 'soda-tester/demo');
  assert.equal(enriched.statusText, 'assessing');
  assert.doesNotMatch(enriched.timeText, /#42/);
});

test('plan, work and attention sections stay honest to entry facts', () => {
  const plan = describePlanIssue(record, repo);
  assert.match(plan.identityText, /#41/);
  assert.equal(plan.issuesLink, '/soda-tester/demo/issues');
  const work = describeWorkEvidence(record, repo);
  assert.match(work.provenanceText, /#41/);
  assert.equal(work.sourceLinks.length, 1);
  assert.equal(describeDecisionAttention(record, false).tone, 'quiet');
  assert.equal(describeDecisionAttention(record, true).tone, 'notice');
  assert.equal(describeDecisionAttention(post, false).tone, 'quiet');
  assert.equal(describePostBody({state: 'loading'}).bodyText, 'Loading discussion text…');
  assert.match(describePostBody({state: 'text', body: 'hello', edited: true}).editedText, /edited/);
});

test('discussion submits real drafts with stored outcomes beside them', () => {
  const discussion = describeDiscussion(post, repo, 'half-written reply', []);
  assert.equal(discussion.draft, 'half-written reply');
  assert.equal(discussion.submitAvailable, true);
  assert.equal(describeDiscussion(post, repo, '   ', []).submitAvailable, false);
  assert.equal(discussion.discussionLink, '/soda-tester/demo/issues');
  const full = describeFeedPost(record, repo, '', [], undefined, false);
  assert.equal(full.header.key, 'record:' + 'a'.repeat(32));
  assert.equal(full.postBody, undefined);
  const withBody = describeFeedPost(post, repo, '', [], {state: 'text', body: 'hi', edited: false}, false);
  assert.equal(withBody.postBody?.bodyText, 'hi');
});

test('comment posts, details and outcomes follow the Lane B contract', () => {
  const postBody = commentPostBody({repository: 7, body: 'please fix', idempotencyKey: 'k1'});
  assert.deepEqual(postBody, {repository: 7, body: 'please fix', idempotency_key: 'k1'});
  const reply = commentPostBody({repository: 7, body: 'agree', idempotencyKey: 'k2', replyTo: 'c'.repeat(32)});
  assert.equal(reply.reply_to, 'c'.repeat(32));
  assert.throws(() => commentPostBody({repository: 0, body: 'x', idempotencyKey: 'k'}));
  assert.throws(() => commentPostBody({repository: 7, body: '   ', idempotencyKey: 'k'}));
  assert.throws(() => commentPostBody({repository: 7, body: 'x', idempotencyKey: 'bad key!'}));
  assert.throws(() => commentPostBody({repository: 7, body: 'x', idempotencyKey: 'k', replyTo: 'nope'}));
  assert.match(newIdempotencyKey(), /^[A-Za-z0-9_-]{1,64}$/);
  const pending = parseCommentPipelineState({
    comment_id: 'c'.repeat(32),
    revision: 0,
    intake: 'recorded',
    judgment: 'pending',
    draft: 'pending',
    effect: 'pending',
  });
  assert.equal(commentUnsettled(pending), true);
  assert.equal(commentUnsettled({...pending, judgment: 'failed'}), false);
  assert.equal(commentUnsettled({...pending, judgment: 'applied', route: 'discussion'}), false);
  assert.equal(
    commentUnsettled({
      ...pending,
      judgment: 'applied',
      route: 'new_issue',
      draft: 'applied',
      effect: 'verified',
      issue: 12,
    }),
    false
  );
  const wirePending = {
    comment_id: 'c'.repeat(32),
    revision: 0,
    intake: 'recorded',
    judgment: 'pending',
    draft: 'pending',
    effect: 'pending',
  };
  const detail = parseCommentDetail({...wirePending, body: 'hello', author_name: 'Ada', at: 5, edited: false});
  assert.equal(detail.body, 'hello');
  assert.equal(detail.removed, false);
  assert.equal(parseCommentDetail({...wirePending, body: '', deleted: true}).removed, true);
  const waiting = describeCommentOutcome(pending, repo);
  assert.equal(waiting.tone, 'progress');
  assert.equal(waiting.retryable, true);
  const created = describeCommentOutcome(
    {...pending, judgment: 'applied', route: 'new_issue', draft: 'applied', effect: 'verified', issue: 12},
    repo
  );
  assert.equal(created.tone, 'settled');
  assert.equal(created.issueLink, '/soda-tester/demo/issues/12');
  assert.equal(describeCommentOutcome({...pending, judgment: 'applied', route: 'discussion'}, repo).retryable, false);
  assert.match(describeCommentOutcome({...pending, judgment: 'applied', route: 'clarify'}, repo).text, /clarification/);
});

function roleRecord(actor: string, status?: string): FeedEntry {
  const entry: FeedEntry = {kind: 'record', id: 'b'.repeat(32), repository: 7, seq: 50, actor};
  if (status !== undefined) entry.status = status;
  return entry;
}

test('every factory-loop role selects its registered feed family', () => {
  assert.equal(selectFeedFamily(roleRecord('product_lead')), 'plan');
  assert.equal(selectFeedFamily(roleRecord('Product Lead')), 'plan');
  assert.equal(selectFeedFamily(roleRecord('architect')), 'issue');
  assert.equal(selectFeedFamily(roleRecord('builder')), 'work');
  assert.equal(selectFeedFamily(roleRecord('inspector')), 'review');
  assert.equal(selectFeedFamily(roleRecord('fixer')), 'correction');
  assert.equal(selectFeedFamily(roleRecord('strategist')), 'reassessment');
  assert.equal(selectFeedFamily(roleRecord('test_pilot')), 'verification');
  assert.equal(selectFeedFamily(roleRecord('library_admin')), 'knowledge');
  assert.equal(selectFeedFamily(roleRecord('jev')), 'decision');
  assert.equal(selectFeedFamily(roleRecord('factory')), 'integration');
  assert.equal(
    selectFeedFamily({kind: 'record', id: 'x', repository: 7, seq: 1, status: 'question_pending'}),
    'question'
  );
  assert.equal(selectFeedFamily(post), 'discussion');
  assert.equal(selectFeedFamily(roleRecord('unknown_actor')), 'fallback');
  assert.equal(selectFeedFamily(record), 'fallback');
  assert.equal(roleLabel('test_pilot'), 'Test Pilot');
  assert.equal(roleLabel('Ada (human reviewer)'), 'Ada (human reviewer)');
  assert.equal(describePostHeader(roleRecord('inspector'), repo).role, 'Inspector');
});

test('recorded statuses classify without ever upgrading to success', () => {
  assert.equal(classifyFeedStatus('check_failed'), 'failure');
  assert.equal(classifyFeedStatus('blocked'), 'failure');
  assert.equal(classifyFeedStatus('stuck'), 'failure');
  assert.equal(classifyFeedStatus('stale'), 'failure');
  assert.equal(classifyFeedStatus('submitted'), 'waiting');
  assert.equal(classifyFeedStatus('assessing'), 'waiting');
  assert.equal(classifyFeedStatus('verified'), 'settled');
  assert.equal(classifyFeedStatus('merged'), 'settled');
  assert.equal(classifyFeedStatus('frobnicate'), 'unknown');
  assert.equal(classifyFeedStatus(undefined), 'unknown');
  assert.equal(classifyFeedStatus(''), 'unknown');
});

test('family sections carry designed anatomy with honest state and links', () => {
  const review = describeFeedFamily(roleRecord('inspector', 'findings_submitted'), repo);
  assert.equal(review.label, 'Independent review');
  assert.match(review.headline, /soda-tester\/demo/);
  assert.match(review.detailText, /coverage/i);
  assert.match(review.stateText, /findings_submitted/);
  assert.equal(review.sourceLinks.length, 2);
  const verification = describeFeedFamily(roleRecord('test_pilot', 'verified'), repo);
  assert.equal(verification.statusClass, 'settled');
  assert.match(verification.detailText, /never exercised/);
  const correction = describeFeedFamily(roleRecord('fixer', 'submitted'), repo);
  assert.match(correction.detailText, /independent acceptance/);
  const reassessment = describeFeedFamily(roleRecord('strategist', 'proposed'), repo);
  assert.match(reassessment.detailText, /not the current graph/);
  const integration = describeFeedFamily(roleRecord('factory', 'merged'), repo);
  assert.match(integration.detailText, /not deployed/);
  const decision = describeFeedFamily(roleRecord('jev', 'recorded'), repo);
  assert.match(decision.detailText, /does not by itself apply/);
});

test('failures lead and malformed enrichment degrades to unknown, never success', () => {
  const failed = roleRecord('builder', 'check_failed');
  const family = describeFeedFamily(failed, repo);
  assert.equal(family.statusClass, 'failure');
  assert.match(family.nextText, /needs attention/);
  const attention = describeDecisionAttention(failed, false);
  assert.equal(attention.tone, 'notice');
  assert.match(attention.attentionText, /check_failed/);
  const unknown = describeDecisionAttention(roleRecord('builder', 'frobnicate'), false);
  assert.equal(unknown.tone, 'notice');
  assert.match(unknown.attentionText, /unknown, not successful/);
  const bare = describeFeedFamily(record, repositoryRef(11, undefined, undefined, undefined));
  assert.equal(bare.family, 'fallback');
  assert.equal(bare.statusClass, 'unknown');
  assert.match(bare.stateText, /unknown, not successful/);
  assert.equal(bare.sourceLinks.length, 0);
  const post = describeFeedPost(roleRecord('inspector', 'check_failed'), repo, '', [], undefined, false);
  assert.equal(post.family.family, 'review');
  assert.equal(post.attention.tone, 'notice');
});

test('family sections render bound issue, PR, candidate reference and recorded summary', () => {
  const candidate = 'c'.repeat(40);
  const entry: FeedEntry = {
    kind: 'record',
    id: 'b'.repeat(32),
    repository: 7,
    seq: 50,
    actor: 'builder',
    status: 'check_failed',
    issue: 3,
    pr: 42,
    ref: candidate,
    summary: 'built the candidate <exactly>',
  };
  const family = describeFeedFamily(entry, repo);
  assert.equal(family.summaryText, 'Recorded: built the candidate <exactly>');
  assert.equal(family.refText, `Candidate reference: ${candidate}.`);
  // Failures still lead: recorded content never upgrades the state.
  assert.equal(family.statusClass, 'failure');
  assert.match(family.stateText, /check_failed/);
  assert.match(family.nextText, /needs attention/);
  // Bound records link natively, replacing the generic indexes.
  assert.deepEqual(
    family.sourceLinks.map((link) => link.href),
    ['/soda-tester/demo', '/soda-tester/demo/issues/3', '/soda-tester/demo/pulls/42']
  );
  assert.deepEqual(
    family.sourceLinks.map((link) => link.label),
    ['Open soda-tester/demo', 'Open issue #3', 'Open pull request #42']
  );
  assert.equal(describeFeedPost(entry, repo, '', [], undefined, false).family.summaryText, family.summaryText);
  // A bound issue replaces the issue index on issue families too.
  const planEntry: FeedEntry = {
    kind: entry.kind,
    id: entry.id,
    repository: entry.repository,
    seq: entry.seq,
    actor: 'product_lead',
    issue: 3,
  };
  const plan = describeFeedFamily(planEntry, repo);
  assert.deepEqual(
    plan.sourceLinks.map((link) => link.href),
    ['/soda-tester/demo', '/soda-tester/demo/issues/3']
  );
});

test('recorded status segments split back into phase, counts and chain', () => {
  const review = parseFeedStatusSegments('review:finished:succeeded:request-changes>findings:3');
  assert.equal(review.phase, 'review');
  assert.equal(review.stage, 'finished');
  assert.equal(review.outcome, 'succeeded');
  assert.equal(review.verdict, 'request-changes');
  assert.equal(review.findings, 3);
  assert.deepEqual(review.chain, []);
  const verify = parseFeedStatusSegments(
    'verify:finished:succeeded:verified>checks:12>failures:2>published>merged>completed'
  );
  assert.equal(verify.checks, 12);
  assert.equal(verify.failures, 2);
  assert.deepEqual(verify.chain, ['published', 'merged', 'completed']);
  // Uncomposed or missing statuses keep no segments rather than guessing.
  for (const raw of [undefined, '', 'check_failed', 'submitted', ':broken']) {
    const segments = parseFeedStatusSegments(raw);
    assert.equal(segments.phase, undefined);
    assert.equal(segments.findings, undefined);
    assert.deepEqual(segments.chain, []);
  }
});

test('designed family blocks are structured per role from recorded facts', () => {
  const review = describeFeedFamily(
    {
      ...roleRecord('inspector'),
      status: 'review:finished:succeeded:request-changes>findings:3',
      summary: 'Found a permission gap.',
    },
    repo
  );
  assert.deepEqual(
    review.blocks.map((block) => block.kind),
    ['brief', 'rows']
  );
  const reviewRows = review.blocks[1];
  assert.equal(reviewRows?.kind, 'rows');
  if (reviewRows?.kind === 'rows') {
    assert.match(reviewRows.rows.find((row) => row.term === 'Verdict')?.value ?? '', /request changes/);
    assert.match(reviewRows.rows.find((row) => row.term === 'Findings')?.value ?? '', /3 recorded findings/);
  }
  const correction = describeFeedFamily(
    {...roleRecord('fixer'), status: 'correct:finished:succeeded>findings:2', ref: 'd'.repeat(40)},
    repo
  );
  const beforeAfter = correction.blocks.find((block) => block.kind === 'beforeAfter');
  assert.equal(beforeAfter?.kind, 'beforeAfter');
  if (beforeAfter?.kind === 'beforeAfter') {
    assert.match(beforeAfter.before, /2 recorded findings/);
    assert.match(beforeAfter.after, /independent acceptance/);
  }
  const verification = describeFeedFamily(
    {...roleRecord('test_pilot'), status: 'verify:finished:succeeded:verified>checks:12>failures:2'},
    repo
  );
  const checks = verification.blocks.find((block) => block.kind === 'checks');
  assert.equal(checks?.kind, 'checks');
  if (checks?.kind === 'checks')
    assert.match(checks.rows.find((row) => row.name.startsWith('Passed'))?.result ?? '', /^10$/);
  const integration = describeFeedFamily(
    {...roleRecord('factory'), status: 'implement:finished:succeeded>published>merged', ref: 'c'.repeat(40)},
    repo
  );
  const chain = integration.blocks.find((block) => block.kind === 'steps');
  assert.equal(chain?.kind, 'steps');
  if (chain?.kind === 'steps') {
    assert.equal(chain.steps.find((step) => step.label === 'merged')?.state, 'done');
    assert.equal(chain.steps.find((step) => step.label === 'completed')?.state, 'waiting');
  }
  const plan = describeFeedFamily(roleRecord('product_lead', 'submitted'), repo);
  const adoption = plan.blocks.find((block) => block.kind === 'steps');
  assert.equal(adoption?.kind, 'steps');
  if (adoption?.kind === 'steps') assert.match(adoption.steps.at(-1)?.state ?? '', /not adopted work/);
  // Unenriched records keep honest per-row unknowns, never invented counts.
  const bare = describeFeedFamily(roleRecord('builder', 'submitted'), repo);
  const bareChecks = bare.blocks.find((block) => block.kind === 'checks');
  assert.equal(bareChecks?.kind, 'checks');
  if (bareChecks?.kind === 'checks') assert.match(bareChecks.rows[0]?.result ?? '', /not recorded/);
  // Every family renders at least one designed block; fallback stays honest.
  for (const actor of [
    'product_lead',
    'architect',
    'builder',
    'inspector',
    'fixer',
    'strategist',
    'test_pilot',
    'library_admin',
    'jev',
    'factory',
  ])
    assert.ok(describeFeedFamily(roleRecord(actor, 'submitted'), repo).blocks.length >= 1, actor);
});

test('comment source carry, existing-work link and durable pending reads', () => {
  // Commenting from a work card carries its bound source issue.
  assert.equal(commentPostBody({repository: 7, body: 'on this issue', idempotencyKey: 'k', issue: 3}).issue, 3);
  assert.throws(() => commentPostBody({repository: 7, body: 'x', idempotencyKey: 'k', issue: -1}));
  const base = {
    comment_id: 'c'.repeat(32),
    revision: 0,
    intake: 'recorded',
    judgment: 'applied',
    route: 'existing_work',
    draft: 'pending',
    effect: 'pending',
    issue: 0,
  };
  // The persisted selected-issue link may arrive as its own additive field.
  assert.equal(parseCommentPipelineState({...base, existing_issue: 9}).issue, 9);
  assert.equal(parseCommentPipelineState({...base, selected_issue: 11}).issue, 11);
  assert.equal(parseCommentPipelineState({...base, issue: 5, existing_issue: 9}).issue, 5);
  const linking = parseCommentPipelineState(base);
  // A known pending existing-work row stays unsettled so refresh re-reads it.
  assert.equal(commentUnsettled(linking), true);
  const linkingOutcome = describeCommentOutcome(linking, repo);
  assert.equal(linkingOutcome.tone, 'progress');
  assert.equal(linkingOutcome.retryable, true);
  assert.match(linkingOutcome.text, /persisted selected-issue link appears here/);
  const linked = parseCommentPipelineState({...base, effect: 'verified', existing_issue: 9});
  assert.equal(commentUnsettled(linked), false);
  const linkedOutcome = describeCommentOutcome(linked, repo);
  assert.equal(linkedOutcome.tone, 'settled');
  assert.equal(linkedOutcome.issueLink, '/soda-tester/demo/issues/9');
  assert.match(linkedOutcome.text, /persisted selected issue #9/);
  // Pipeline stages read as recorded words, ending at the persisted link.
  assert.deepEqual(
    linkedOutcome.stages.map((stage) => stage.state),
    ['done', 'done', 'done', 'done — issue #9']
  );
  const pendingStages = describeCommentOutcome(
    parseCommentPipelineState({...base, judgment: 'pending', route: undefined}),
    repo
  ).stages;
  assert.deepEqual(
    pendingStages.map((stage) => stage.state),
    ['done', 'pending']
  );
});

test('unenriched records keep generic links and no recorded-content lines', () => {
  const work = describeFeedFamily(roleRecord('builder', 'submitted'), repo);
  assert.equal(work.summaryText, undefined);
  assert.equal(work.refText, undefined);
  assert.deepEqual(
    work.sourceLinks.map((link) => link.href),
    ['/soda-tester/demo', '/soda-tester/demo/pulls']
  );
  // Unusable values degrade to absence, never an invented link or line.
  const odd = describeFeedFamily(
    {kind: 'record', id: 'x', repository: 7, seq: 1, actor: 'builder', issue: 0, pr: 0, ref: '', summary: ''},
    repo
  );
  assert.equal(odd.summaryText, undefined);
  assert.equal(odd.refText, undefined);
  assert.deepEqual(
    odd.sourceLinks.map((link) => link.href),
    ['/soda-tester/demo', '/soda-tester/demo/pulls']
  );
  // Without a native repository path the recorded text still renders,
  // with no fabricated links.
  const unlinked = describeFeedFamily(
    {
      kind: 'record',
      id: 'x',
      repository: 11,
      seq: 1,
      actor: 'builder',
      issue: 3,
      pr: 42,
      ref: 'c'.repeat(40),
      summary: 'built the candidate',
    },
    repositoryRef(11, undefined, undefined, undefined)
  );
  assert.equal(unlinked.sourceLinks.length, 0);
  assert.match(unlinked.summaryText ?? '', /built the candidate/);
  assert.match(unlinked.refText ?? '', /c{40}/);
});

test('milestone entries admit activity and bound assignment without inventing them', () => {
  const merged = parseFeedEntry({
    kind: 'record',
    id: '9'.repeat(32),
    repository: 7,
    seq: 50,
    activity: 'merged',
    assignment: '2'.repeat(32),
    status: 'merged>closed',
  });
  assert.equal(merged.activity, 'merged');
  assert.equal(merged.assignment, '2'.repeat(32));
  const bare = parseFeedEntry({kind: 'record', id: 'x', repository: 7, seq: 1});
  assert.equal(bare.activity, undefined);
  assert.equal(bare.assignment, undefined);
  assert.equal(bare.payload, undefined);
  for (const bad of [
    {kind: 'record', id: 'x', repository: 7, seq: 1, activity: 'shipped'},
    {kind: 'record', id: 'x', repository: 7, seq: 1, activity: ''},
    {kind: 'record', id: 'x', repository: 7, seq: 1, activity: 7},
    {kind: 'record', id: 'x', repository: 7, seq: 1, assignment: ''},
    {kind: 'record', id: 'x', repository: 7, seq: 1, assignment: 'a'.repeat(257)},
  ])
    assert.throws(() => parseFeedEntry(bad));
});

test('typed plan payloads parse their graph and render steps, dependencies and planned checks', () => {
  const entry = parseFeedEntry({
    kind: 'record',
    id: 'b'.repeat(32),
    repository: 7,
    seq: 50,
    actor: 'architect',
    status: 'plan:finished:succeeded',
    payload: {
      plan: {
        approach: 'Split the form service.',
        source_commit: 'c'.repeat(40),
        generation: 2,
        findings: ['The form mixes storage and HTTP.'],
        findings_total: 3,
        unresolved: ['Who owns migrations?'],
        rejected: [{approach: 'Rewrite', reason: 'Too risky'}],
        inspected: [{path: 'internal/form.go', commit: 'c'.repeat(40), evidence: 'Handlers write rows directly.'}],
        inspected_total: 1,
        acceptance_coverage: ['a1'],
        tasks: [
          {id: 't1', objective: 'Extract storage', lane: 'core', outputs: ['storage.go'], checks: ['unit']},
          {id: 't2', objective: 'Wire handlers', depends_on: ['t1']},
        ],
        tasks_total: 30,
        lanes: [{id: 'core', tasks: ['t1']}],
        checks: [{id: 'unit', argv: ['go', 'test', './...'], scope: 'repo', required: true}],
        checks_total: 1,
      },
    },
  });
  assert.equal(entry.payload?.plan?.tasks[1]?.dependsOn.join(','), 't1');
  assert.equal(entry.payload?.plan?.checks[0]?.required, true);
  const family = describeFeedFamily(entry, repo);
  const steps = family.blocks.find((block) => block.kind === 'steps' && block.heading.startsWith('Plan tasks'));
  assert.equal(steps?.kind, 'steps');
  if (steps?.kind === 'steps') {
    assert.match(steps.steps[0]?.state ?? '', /lane core; starts first/);
    assert.match(steps.steps[1]?.state ?? '', /after t1/);
    assert.match(steps.note ?? '', /Showing 2 of 30 recorded plan tasks/);
  }
  const checks = family.blocks.find((block) => block.kind === 'checks' && block.heading.startsWith('Planned checks'));
  assert.equal(checks?.kind, 'checks');
  if (checks?.kind === 'checks') {
    assert.match(checks.rows[0]?.name ?? '', /go test/);
    assert.match(checks.rows[0]?.result ?? '', /required; scope repo; planned only — not run/);
  }
  const findings = family.blocks.find((block) => block.kind === 'rows' && block.heading.startsWith('Investigation'));
  assert.equal(findings?.kind, 'rows');
  if (findings?.kind === 'rows') assert.match(findings.note ?? '', /Showing 1 of 3 recorded investigation findings/);
});

test('correction payloads join finding bodies to dispositions and keep orphans honest', () => {
  const entry = parseFeedEntry({
    kind: 'record',
    id: 'b'.repeat(32),
    repository: 7,
    seq: 50,
    actor: 'fixer',
    status: 'correct:finished:succeeded>findings:2',
    payload: {
      tasks: {tasks: [{id: 't1', state: 'done'}], total: 1},
      correction: {
        new_candidate: 'd'.repeat(40),
        set_digest: 'set-1',
        findings: [
          {
            key: 'f1',
            severity: 'high',
            defect: 'Cross-team read.',
            evidence: 'Query lacks team scope.',
            required_outcome: 'Scope by team.',
            fix_kind: 'code',
            actionable: true,
            location: {path: 'store.go', side: 'right', line: 42},
          },
          {key: 'f2', defect: 'Missing test.'},
        ],
        findings_total: 2,
        dispositions: [
          {key: 'f1', disposition: 'fixed', changed_paths: ['store.go'], commits: ['e'.repeat(40)]},
          {key: 'ghost', disposition: 'wontfix'},
        ],
        dispositions_total: 2,
      },
    },
  });
  // Tasks may accompany a correction; the pairing survives parsing.
  assert.equal(entry.payload?.tasks?.tasks[0]?.state, 'done');
  const family = describeFeedFamily(entry, repo);
  const joined = family.blocks.find((block) => block.kind === 'rows' && block.heading.startsWith('Finding dispositions'));
  assert.equal(joined?.kind, 'rows');
  if (joined?.kind === 'rows') {
    assert.equal(joined.rows.length, 3);
    assert.match(joined.rows[0]?.term ?? '', /f1 \(high\)/);
    assert.match(joined.rows[0]?.value ?? '', /Defect: Cross-team read\./);
    assert.match(joined.rows[0]?.value ?? '', /At store\.go line 42 \(right side\)\./);
    assert.match(joined.rows[0]?.value ?? '', /Disposition: fixed\. Changed paths: store\.go\. Commits: eeeeeeeeeeee…\./);
    assert.match(joined.rows[1]?.value ?? '', /No disposition recorded\./);
    assert.match(joined.rows[2]?.value ?? '', /Finding body not carried on this entry\./);
  }
});

test('verification payloads lead with failures, then real receipts and coverage', () => {
  const entry = parseFeedEntry({
    kind: 'record',
    id: 'b'.repeat(32),
    repository: 7,
    seq: 50,
    actor: 'test_pilot',
    status: 'verify:finished:succeeded:verified>checks:2>failures:1',
    payload: {
      verification: {
        outcome: 'verified',
        delivery_id: 'delivery-1',
        checks: [{id: 'login', outcome: 'passed', artifact_digest: 'f'.repeat(40)}],
        checks_total: 9,
        failures: [{check: 'export', detail: 'Export timed out.'}],
        coverage: [{intent_id: 'i1', outcome_id: 'o1', task_keys: ['t1'], delivered_acceptance_ids: ['a1']}],
        coverage_total: 1,
        artifacts_total: 4,
        evidence_total: 6,
      },
    },
  });
  const headings = describeFeedFamily(entry, repo).blocks.map((block) => block.heading);
  assert.ok(headings.indexOf('Recorded failures (TP11)') < headings.indexOf('Check receipts (TP07)'));
  assert.ok(headings.indexOf('Check receipts (TP07)') < headings.indexOf('Accepted-outcome coverage (TP02)'));
  const receipts = describeFeedFamily(entry, repo).blocks.find(
    (block) => block.kind === 'checks' && block.heading === 'Check receipts (TP07)'
  );
  assert.equal(receipts?.kind, 'checks');
  if (receipts?.kind === 'checks') {
    assert.match(receipts.rows[0]?.result ?? '', /passed; artifact ffffffffffff…/);
    assert.match(receipts.note ?? '', /Showing 1 of 9 recorded check receipts/);
  }
});

test('judgment payloads render the selection, candidates and honest totals', () => {
  const entry = parseFeedEntry({
    kind: 'record',
    id: 'ab'.repeat(16),
    repository: 7,
    seq: 50,
    actor: 'jev',
    activity: 'judgment',
    assignment: '2'.repeat(32),
    status: 'judgment:priority:selected',
    payload: {
      judgment: {
        kind: 'priority',
        selected: 'cand-a',
        outcome: 'selected',
        model: 'jev',
        confidence: 0.7,
        probabilities: {'cand-a': 0.6, 'cand-b': 0.4},
        candidates: [
          {id: 'cand-a', repository: 7, issue: 3, objective: 'Ship it'},
          {id: 'cand-b', repository: 7, issue: 4, objective: 'Later'},
        ],
        candidates_total: 30,
      },
    },
  });
  assert.equal(selectFeedFamily(entry), 'decision');
  assert.match(milestoneText(entry) ?? '', /Milestone: priority judgment · on assignment 222222222222…/);
  assert.match(describePostHeader(entry, repo).milestoneText ?? '', /priority judgment/);
  assert.match(describePlanIssue(entry, repo).identityText, /Judgment abababababab… on assignment 222222222222…/);
  const family = describeFeedFamily(entry, repo);
  const record = family.blocks.find((block) => block.kind === 'rows' && block.heading === 'Recorded judgment (F04)');
  assert.equal(record?.kind, 'rows');
  if (record?.kind === 'rows') {
    assert.equal(record.rows.find((row) => row.term === 'Selected')?.value, 'cand-a');
    assert.equal(record.rows.find((row) => row.term === 'Confidence')?.value, '70%');
  }
  const candidates = family.blocks.find(
    (block) => block.kind === 'checks' && block.heading === 'Candidates considered (F04)'
  );
  assert.equal(candidates?.kind, 'checks');
  if (candidates?.kind === 'checks') {
    assert.match(candidates.rows[0]?.name ?? '', /cand-a — Ship it — selected/);
    assert.match(candidates.rows[0]?.result ?? '', /60%; issue #3 in repository 7/);
    assert.match(candidates.note ?? '', /Showing 2 of 30 recorded candidates/);
  }
});

test('merge and completion milestones are their own families and steps', () => {
  const merged = parseFeedEntry({
    kind: 'record',
    id: '9'.repeat(32),
    repository: 7,
    seq: 50,
    actor: 'factory',
    activity: 'merged',
    assignment: '2'.repeat(32),
    status: 'merged>closed',
    issue: 3,
    pr: 42,
    ref: 'e'.repeat(40),
    payload: {
      merge: {
        stage: 'merged',
        publication: '9'.repeat(32),
        merged_commit: 'e'.repeat(40),
        merged_unix: 1700002000,
        original_closed_unix: 1700002100,
      },
    },
  });
  assert.equal(selectFeedFamily(merged), 'merged');
  const mergeFamily = describeFeedFamily(merged, repo);
  assert.equal(mergeFamily.statusClass, 'settled');
  const mergeSteps = mergeFamily.blocks.find((block) => block.kind === 'steps' && block.heading === 'Merge (F06)');
  assert.equal(mergeSteps?.kind, 'steps');
  if (mergeSteps?.kind === 'steps') {
    assert.match(mergeSteps.steps[0]?.state ?? '', /done — merged commit e{40}/);
    assert.match(mergeSteps.steps[1]?.state ?? '', /^done — /);
  }
  const completed = parseFeedEntry({
    kind: 'record',
    id: 'e'.repeat(32),
    repository: 7,
    seq: 50,
    actor: 'factory',
    activity: 'completed',
    assignment: '2'.repeat(32),
    status: 'completed',
    payload: {
      completion: {
        event: 'e'.repeat(32),
        publication: '9'.repeat(32),
        occurrence: 'occ-1',
        merged_commit: 'e'.repeat(40),
        generation: 1,
      },
    },
  });
  assert.equal(selectFeedFamily(completed), 'completed');
  const completionSteps = describeFeedFamily(completed, repo).blocks.find(
    (block) => block.kind === 'steps' && block.heading === 'Verified completion (F07/F10)'
  );
  assert.equal(completionSteps?.kind, 'steps');
  // Independent recording order wins over any assumed lifecycle ranking.
  completed.seq = 55;
  merged.seq = 54;
  const delivery: FeedEntry = {
    kind: 'record',
    id: 'd'.repeat(32),
    repository: 7,
    seq: 53,
    activity: 'judgment',
    status: 'judgment:delivery:ready',
  };
  const assignment: FeedEntry = {kind: 'record', id: '2'.repeat(32), repository: 7, seq: 52};
  const priority: FeedEntry = {
    kind: 'record',
    id: 'cd'.repeat(16),
    repository: 7,
    seq: 50,
    activity: 'judgment',
    status: 'judgment:priority:selected',
  };
  assert.deepEqual(
    mergeFeedEntries([], [priority, assignment, merged, completed, delivery]).map((entry) => entry.id),
    [completed.id, merged.id, delivery.id, assignment.id, priority.id]
  );
});

test('reassessment and implementation pairings render before/after and task states', () => {
  const reassessment = parseFeedEntry({
    kind: 'record',
    id: 'b'.repeat(32),
    repository: 7,
    seq: 50,
    actor: 'strategist',
    status: 'reassess:finished:succeeded',
    payload: {
      reassessment: {
        retained_issues: [3, 5],
        changes: [{intent_id: 'i1', reason: 'Stale plan', tasks: [{key: 't9', objective: 'Redo wiring'}]}],
        changes_total: 1,
        attention: [{issue: 5, reason: 'Blocked on review'}],
        coverage: [],
        inspected_total: 12,
      },
    },
  });
  const family = describeFeedFamily(reassessment, repo);
  const panels = family.blocks.filter((block) => block.kind === 'beforeAfter');
  assert.ok(panels.length >= 1);
  const typed = panels[0];
  assert.equal(typed?.kind, 'beforeAfter');
  if (typed?.kind === 'beforeAfter') {
    assert.match(typed.before, /Retained issues: #3, #5/);
    assert.match(typed.after, /i1: Stale plan/);
  }
  const implementation = parseFeedEntry({
    kind: 'record',
    id: 'b'.repeat(32),
    repository: 7,
    seq: 51,
    actor: 'builder',
    status: 'implement:finished:succeeded',
    payload: {
      implementation: {candidate: 'c'.repeat(40), diagnostic_refs: ['diag-1']},
      tasks: {tasks: [{id: 't1', state: 'done'}, {id: 't2'}], total: 2},
    },
  });
  const states = describeFeedFamily(implementation, repo).blocks.find(
    (block) => block.kind === 'steps' && block.heading.startsWith('Recorded task states')
  );
  assert.equal(states?.kind, 'steps');
  if (states?.kind === 'steps') assert.equal(states.steps[1]?.state, 'not recorded');
});

test('malformed payloads degrade to an honest alert, never a coerced success', () => {
  for (const payload of [
    'typed',
    null,
    {plan: 'x'},
    {future_family: {}},
    {plan: {}, review: {}},
    {tasks: {tasks: []}},
    {tasks: {tasks: []}, judgment: {kind: 'priority'}},
    {judgment: {kind: 'priority', probability: 2}},
    {judgment: {kind: 'coin-flip'}},
    {plan: {tasks: [{}]}},
    {plan: {tasks: [{id: 't1'}], tasks_total: 0}},
    {verification: {checks: [{id: 'c1', outcome: 5}]}},
    {review: {new_findings: [{defect: 'no key'}]}},
  ]) {
    const entry = parseFeedEntry({
      kind: 'record',
      id: 'b'.repeat(32),
      repository: 7,
      seq: 50,
      actor: 'builder',
      status: 'check_failed',
      payload,
    });
    assert.deepEqual(entry.payload, {invalid: true});
    const family = describeFeedFamily(entry, repo);
    // The recorded failure still leads; the unreadable detail is an
    // alert block, and nothing renders as done.
    assert.equal(family.statusClass, 'failure');
    assert.equal(family.blocks[0]?.kind, 'alert');
    if (family.blocks[0]?.kind === 'alert') assert.match(family.blocks[0].text, /could not be read/);
  }
  assert.equal(parseFeedPayload(undefined), undefined);
  assert.equal(parseFeedPayload({}), undefined);
  assert.equal(
    parseFeedEntry({kind: 'record', id: 'x', repository: 7, seq: 1, payload: {}}).payload,
    undefined
  );
  // A bare tasks preview with no role family is not a valid payload.
  assert.deepEqual(parseFeedPayload({tasks: {tasks: [{id: 't1'}]}}), {invalid: true});
});

test('pages remain bounded when milestones have independent order keys', () => {
  assert.equal(feedPageMaxEntries, 256);
  const entries = Array.from({length: feedPageMaxEntries}, (_, index) => ({
    kind: 'record', id: 'entry-' + index, repository: 7, seq: 1000 - index,
  }));
  const page = parseFeedPageView({entries, next: 'v1:745', truncated: true});
  assert.equal(page.entries.length, 256);
  assert.throws(() => parseFeedPageView({entries: [...entries, {...entries[0], id: 'extra'}], truncated: true}));
});

test('comment submissions name their source entry for server resolution', () => {
  // Records name their assignment; milestone posts name the bound
  // assignment, never the milestone's own publication/event identity.
  assert.deepEqual(commentSourceForEntry({kind: 'record', id: 'a'.repeat(32), repository: 7, seq: 1}), {
    kind: 'record',
    id: 'a'.repeat(32),
  });
  assert.deepEqual(
    commentSourceForEntry({
      kind: 'record',
      id: '9'.repeat(32),
      repository: 7,
      seq: 1,
      activity: 'merged',
      assignment: '2'.repeat(32),
    }),
    {kind: 'record', id: '2'.repeat(32)}
  );
  assert.deepEqual(commentSourceForEntry({kind: 'post', id: 'c'.repeat(32), repository: 7, seq: 1}), {
    kind: 'post',
    id: 'c'.repeat(32),
  });
  assert.equal(commentSourceForEntry({kind: 'record', id: 'not-hex', repository: 7, seq: 1}), undefined);
  const withSource = commentPostBody({
    repository: 7,
    body: 'on this work',
    idempotencyKey: 'k',
    issue: 3,
    source: {kind: 'record', id: 'a'.repeat(32)},
  });
  assert.equal(withSource.source_kind, 'record');
  assert.equal(withSource.source_id, 'a'.repeat(32));
  assert.equal(withSource.issue, 3);
  assert.throws(() =>
    commentPostBody({repository: 7, body: 'x', idempotencyKey: 'k', source: {kind: 'record', id: 'nope'}})
  );
});

test('comment responses carry existing_issue and the resolved source snapshot', () => {
  const state = parseCommentPipelineState({
    comment_id: 'c'.repeat(32),
    revision: 0,
    intake: 'recorded',
    judgment: 'applied',
    route: 'existing_work',
    draft: 'pending',
    effect: 'verified',
    issue: 0,
    existing_issue: 9,
    source: {
      kind: 'record',
      id: 'a'.repeat(32),
      issue: 3,
      pr: 42,
      ref: 'e'.repeat(40),
      status: 'implement:finished:succeeded',
      summary: 'Built the form service.',
    },
  });
  assert.equal(state.existingIssue, 9);
  assert.equal(state.issue, 9);
  assert.equal(state.source?.summary, 'Built the form service.');
  const outcome = describeCommentOutcome(state, repo);
  assert.equal(outcome.tone, 'settled');
  // The existing-work link renders from existing_issue.
  assert.equal(outcome.issueLink, '/soda-tester/demo/issues/9');
  assert.match(outcome.text, /persisted selected issue #9/);
  assert.match(outcome.sourceText, /^About recorded work aaaaaaaaaaaa… · issue #3 · pull request #42 · candidate e{40}/);
  assert.match(outcome.sourceText, /— Built the form service\.$/);
  // Null and absent sources stay absent; malformed ones refuse.
  const bare = parseCommentPipelineState({
    comment_id: 'c'.repeat(32),
    revision: 0,
    intake: 'recorded',
    judgment: 'pending',
    draft: 'pending',
    effect: 'pending',
    source: null,
  });
  assert.equal(bare.source, undefined);
  assert.equal(describeCommentOutcome(bare, repo).sourceText, '');
  assert.throws(() =>
    parseCommentPipelineState({
      comment_id: 'c'.repeat(32),
      revision: 0,
      intake: 'recorded',
      judgment: 'pending',
      draft: 'pending',
      effect: 'pending',
      source: {kind: 'tweet', id: 'a'.repeat(32)},
    })
  );
});

test('exhausted authoring stops polling and reports failure without claiming a native outcome', () => {
  const base = {comment_id: 'a'.repeat(32), revision: 0, intake: 'recorded', judgment: 'applied', route: 'new_issue', draft: 'pending', effect: 'pending', attempts: 6, attempt_limit: 6};
  const repo = {id: 2, path: '/tester/project', label: 'Test project'};
  const exhausted = parseCommentPipelineState(base);
  const outcome = describeCommentOutcome(exhausted, repo);
  assert.equal(commentUnsettled(exhausted), false);
  assert.equal(outcome.tone, 'failed');
  assert.equal(outcome.retryable, false);
  assert.match(outcome.text, /authoring stopped/);
  assert.ok(outcome.stages.some(stage => stage.label === 'Native issue creation' && stage.state === 'not started'));
  assert.equal(commentUnsettled(parseCommentPipelineState({...base, attempts: 5})), true);
  const unknown = describeCommentOutcome(parseCommentPipelineState({...base, draft: 'applied', effect: 'failed'}), repo);
  assert.match(unknown.text, /unconfirmed/);
  assert.doesNotMatch(unknown.text, /nothing was created/);
  const verified = describeCommentOutcome(parseCommentPipelineState({...base, draft: 'applied', effect: 'verified', issue: 9}), repo);
  assert.equal(verified.tone, 'settled');
  assert.equal(verified.issueLink, '/tester/project/issues/9');
});
